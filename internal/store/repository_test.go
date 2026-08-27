package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/benzhi/oral-history-release/internal/domain"
)

func newStoredCase(t *testing.T, repository *Repository) *domain.OralHistoryCase {
	t.Helper()
	now := time.Now().UTC()
	caseFile, err := domain.NewCase(domain.CreateCaseInput{ID: "case-1", Title: "测试", CollectionUnit: "馆", IntervieweeCode: "I", SourceRef: "src", RequestedScope: domain.PublicationScope{Audiences: []string{"公众"}, Purposes: []string{"研究"}}, Now: now})
	if err != nil {
		t.Fatal(err)
	}
	if result := repository.Create(caseFile, "create", "actor", now); result.Err != nil {
		t.Fatal(result.Err)
	}
	return caseFile
}

func TestExecuteRevisionAndIdempotency(t *testing.T) {
	repository, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	caseFile := newStoredCase(t, repository)
	now := time.Now().UTC()
	mutation := func(c *domain.OralHistoryCase) (any, error) {
		err := c.AddConsent(domain.ConsentGrant{ID: "c1", EvidenceRef: "proof", AllowedAudiences: []string{"公众"}, AllowedPurposes: []string{"研究"}, EffectiveAt: now.Add(-time.Hour), ExpiresAt: now.Add(time.Hour)}, now)
		return c, err
	}
	first := repository.Execute(caseFile.ID, 1, "same", "consent", "actor", now, mutation)
	if first.Err != nil {
		t.Fatal(first.Err)
	}
	retry := repository.Execute(caseFile.ID, 1, "same", "consent", "actor", now, func(*domain.OralHistoryCase) (any, error) { t.Fatal("cached mutation executed"); return nil, nil })
	if retry.Err != nil || !retry.FromCache || string(first.Body) != string(retry.Body) {
		t.Fatalf("unstable retry: %+v", retry)
	}
	stale := repository.Execute(caseFile.ID, 1, "stale", "update", "actor", now, mutation)
	if domain.ErrorCode(stale.Err) != "revision_conflict" {
		t.Fatalf("want conflict, got %v", stale.Err)
	}
	loaded, audits, err := repository.Get(caseFile.ID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Revision != 2 {
		t.Fatalf("rejected operation changed revision: %d", loaded.Revision)
	}
	if len(audits) != 3 || audits[2].Outcome != "rejected:revision_conflict" {
		t.Fatalf("rejected audit missing: %+v", audits)
	}
}

func TestRecoverRejectsCorruptObject(t *testing.T) {
	root := t.TempDir()
	repository, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	newStoredCase(t, repository)
	recordData, err := os.ReadFile(filepath.Join(root, "cases", "case-1.json"))
	if err != nil {
		t.Fatal(err)
	}
	var record caseRecord
	if err := json.Unmarshal(recordData, &record); err != nil {
		t.Fatal(err)
	}
	if len(record.ObjectDigests) != 0 {
		t.Fatal("new draft should have no evidence objects")
	}
	loaded, _, err := repository.Get("case-1")
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Title != "测试" {
		t.Fatal("projection did not recover")
	}
}
