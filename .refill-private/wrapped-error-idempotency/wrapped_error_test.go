package wrapped_error_idempotency

import (
	"testing"
	"time"

	"github.com/benzhi/oral-history-release/internal/domain"
	"github.com/benzhi/oral-history-release/internal/store"
)

func TestWrappedMutationErrorPreservesDomainCode(t *testing.T) {
	repository, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 27, 10, 0, 0, 0, time.UTC)
	caseFile, err := domain.NewCase(domain.CreateCaseInput{
		ID: "wrapped-error-case", Title: "错误链测试", CollectionUnit: "档案馆",
		IntervieweeCode: "INT-ERR", SourceRef: "archive://error-chain",
		RequestedScope: domain.PublicationScope{Audiences: []string{"公众"}, Purposes: []string{"研究"}}, Now: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result := repository.Create(caseFile, "create", "整理员", now); result.Err != nil {
		t.Fatal(result.Err)
	}

	mutationCalls := 0
	mutation := func(*domain.OralHistoryCase) (any, error) {
		mutationCalls++
		return nil, domain.ErrConsentBlocked
	}
	first := repository.Execute(caseFile.ID, 1, "blocked-retry", "submit_transcript", "整理员", now, mutation)
	retry := repository.Execute(caseFile.ID, 1, "blocked-retry", "submit_transcript", "整理员", now, func(*domain.OralHistoryCase) (any, error) {
		t.Fatal("幂等重试不应再次执行 mutation")
		return nil, nil
	})

	if got := domain.ErrorCode(first.Err); got != "consent_blocked" {
		t.Errorf("首次失败丢失领域错误码: got %q, err=%v", got, first.Err)
	}
	if got := domain.ErrorCode(retry.Err); got != "consent_blocked" || !retry.FromCache {
		t.Errorf("幂等重试未保留领域错误链: code=%q cached=%v err=%v", got, retry.FromCache, retry.Err)
	}
	if mutationCalls != 1 {
		t.Errorf("mutation 执行次数 = %d, want 1", mutationCalls)
	}
}
