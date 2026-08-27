package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/benzhi/oral-history-release/internal/domain"
)

type Repository struct {
	root        string
	locks       caseLocks
	objectMu    sync.Mutex
	knownObject map[string]struct{}
}

func Open(root string) (*Repository, error) {
	if strings.TrimSpace(root) == "" {
		return nil, fmt.Errorf("数据目录不能为空")
	}
	r := &Repository{root: filepath.Clean(root), knownObject: make(map[string]struct{})}
	for _, dir := range []string{"cases", "objects"} {
		if err := os.MkdirAll(filepath.Join(r.root, dir), 0o750); err != nil {
			return nil, err
		}
	}
	if err := r.Recover(); err != nil {
		return nil, err
	}
	return r, nil
}

func (r *Repository) casePath(caseID string) string {
	return filepath.Join(r.root, "cases", caseID+".json")
}

func validateID(value string) error {
	if value == "" || strings.ContainsAny(value, `/\\`) || value == "." || value == ".." {
		return domain.Invalid("id", "仅允许不含路径分隔符的标识")
	}
	return nil
}

func (r *Repository) readRecord(caseID string) (*caseRecord, error) {
	if err := validateID(caseID); err != nil {
		return nil, err
	}
	data, err := os.ReadFile(r.casePath(caseID))
	if errors.Is(err, os.ErrNotExist) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	var record caseRecord
	if err := json.Unmarshal(data, &record); err != nil {
		return nil, fmt.Errorf("解析案卷 %s: %w", caseID, err)
	}
	if record.SchemaVersion != schemaVersion || record.Aggregate.ID != caseID {
		return nil, fmt.Errorf("案卷 %s schemaVersion 或标识不匹配", caseID)
	}
	if record.Idempotency == nil {
		record.Idempotency = make(map[string]storedResult)
	}
	return &record, nil
}

func (r *Repository) writeRecord(record *caseRecord) error {
	digests, err := r.persistObjects(&record.Aggregate)
	if err != nil {
		return err
	}
	sort.Strings(digests)
	record.ObjectDigests = digests
	record.SchemaVersion = schemaVersion
	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return err
	}
	return atomicWrite(r.casePath(record.Aggregate.ID), data, 0o640)
}

func cloneCase(c domain.OralHistoryCase) (domain.OralHistoryCase, error) {
	data, err := json.Marshal(c)
	if err != nil {
		return domain.OralHistoryCase{}, err
	}
	var cloned domain.OralHistoryCase
	err = json.Unmarshal(data, &cloned)
	return cloned, err
}

func errorResult(err error) storedResult {
	if e, ok := err.(*domain.Error); ok {
		return storedResult{ErrorCode: e.Code, ErrorMessage: e.Message}
	}
	return storedResult{ErrorCode: "internal_error", ErrorMessage: err.Error()}
}

func restoreResult(stored storedResult, cached bool) Result {
	if stored.ErrorCode != "" {
		return Result{Err: domain.NewError(stored.ErrorCode, stored.ErrorMessage), FromCache: cached}
	}
	return Result{Body: append(json.RawMessage(nil), stored.Body...), FromCache: cached}
}

type Mutation func(*domain.OralHistoryCase) (any, error)

func (r *Repository) Create(c *domain.OralHistoryCase, idempotencyKey, actor string, now time.Time) Result {
	if err := validateID(c.ID); err != nil {
		return Result{Err: err}
	}
	lock := r.locks.forCase(c.ID)
	lock.Lock()
	defer lock.Unlock()
	if _, err := os.Stat(r.casePath(c.ID)); err == nil {
		record, readErr := r.readRecord(c.ID)
		if readErr != nil {
			return Result{Err: readErr}
		}
		if cached, ok := record.Idempotency["create_case:"+idempotencyKey]; ok {
			return restoreResult(cached, true)
		}
		return Result{Err: domain.NewError("case_exists", "案卷已经存在")}
	} else if !os.IsNotExist(err) {
		return Result{Err: err}
	}
	body, err := json.Marshal(c)
	if err != nil {
		return Result{Err: err}
	}
	event := domain.NewAuditEvent(1, c.ID, "create_case", actor, "", domain.Digest(c), "success", now)
	record := &caseRecord{
		SchemaVersion: schemaVersion, Aggregate: *c, Audits: []domain.AuditEvent{event},
		Idempotency: map[string]storedResult{"create_case:" + idempotencyKey: {Body: body}},
	}
	if err := r.writeRecord(record); err != nil {
		return Result{Err: err}
	}
	return Result{Body: body}
}

func (r *Repository) Execute(caseID string, expectedRevision int64, idempotencyKey, action, actor string, now time.Time, mutation Mutation) Result {
	if strings.TrimSpace(idempotencyKey) == "" {
		return Result{Err: domain.Invalid("idempotencyKey", "不能为空")}
	}
	lock := r.locks.forCase(caseID)
	lock.Lock()
	defer lock.Unlock()
	record, err := r.readRecord(caseID)
	if err != nil {
		return Result{Err: err}
	}
	cacheKey := action + ":" + idempotencyKey
	if cached, ok := record.Idempotency[cacheKey]; ok {
		return restoreResult(cached, true)
	}
	previousAudit := ""
	if len(record.Audits) > 0 {
		previousAudit = record.Audits[len(record.Audits)-1].Digest
	}
	appendAudit := func(outcome, payload string) {
		record.Audits = append(record.Audits, domain.NewAuditEvent(
			int64(len(record.Audits)+1), caseID, action, actor, previousAudit, payload, outcome, now,
		))
	}
	if expectedRevision != record.Aggregate.Revision {
		stored := errorResult(domain.ErrRevisionConflict)
		record.Idempotency[cacheKey] = stored
		appendAudit("rejected:"+stored.ErrorCode, domain.Digest(expectedRevision))
		if err := r.writeRecord(record); err != nil {
			return Result{Err: err}
		}
		return restoreResult(stored, false)
	}
	cloned, err := cloneCase(record.Aggregate)
	if err != nil {
		return Result{Err: err}
	}
	value, mutationErr := mutation(&cloned)
	if mutationErr != nil {
		stored := errorResult(mutationErr)
		record.Idempotency[cacheKey] = stored
		appendAudit("rejected:"+stored.ErrorCode, domain.Digest(stored))
		if err := r.writeRecord(record); err != nil {
			return Result{Err: err}
		}
		return restoreResult(stored, false)
	}
	body, err := json.Marshal(value)
	if err != nil {
		return Result{Err: err}
	}
	record.Aggregate = cloned
	record.Idempotency[cacheKey] = storedResult{Body: body}
	appendAudit("success", domain.Digest(value))
	if err := r.writeRecord(record); err != nil {
		return Result{Err: err}
	}
	return Result{Body: body}
}

func (r *Repository) Get(caseID string) (*domain.OralHistoryCase, []domain.AuditEvent, error) {
	lock := r.locks.forCase(caseID)
	lock.Lock()
	defer lock.Unlock()
	record, err := r.readRecord(caseID)
	if err != nil {
		return nil, nil, err
	}
	cloned, err := cloneCase(record.Aggregate)
	if err != nil {
		return nil, nil, err
	}
	audits := append([]domain.AuditEvent(nil), record.Audits...)
	return &cloned, audits, nil
}

func (r *Repository) List() ([]domain.OralHistoryCase, error) {
	entries, err := os.ReadDir(filepath.Join(r.root, "cases"))
	if err != nil {
		return nil, err
	}
	result := make([]domain.OralHistoryCase, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		caseID := strings.TrimSuffix(entry.Name(), ".json")
		caseFile, _, err := r.Get(caseID)
		if err != nil {
			return nil, err
		}
		result = append(result, *caseFile)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].UpdatedAt.After(result[j].UpdatedAt) })
	return result, nil
}
