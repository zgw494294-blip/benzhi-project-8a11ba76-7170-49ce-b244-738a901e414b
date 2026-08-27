package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/benzhi/oral-history-release/internal/domain"
)

type checkCase struct {
	ID          string                     `json:"id"`
	Revision    int64                      `json:"revision"`
	State       domain.CaseState           `json:"state"`
	Transcripts []domain.TranscriptVersion `json:"transcripts"`
	Credentials []domain.ReleaseCredential `json:"credentials"`
}

func runSelfcheck(listener net.Listener, server *http.Server) error {
	serveResult := make(chan error, 1)
	go func() { serveResult <- server.Serve(listener) }()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	baseURL := "http://" + listener.Addr().String()
	client := &http.Client{Timeout: 3 * time.Second}
	if err := waitReady(ctx, client, baseURL+"/healthz"); err != nil {
		return shutdownAfterCheck(server, serveResult, err)
	}
	now := time.Now().UTC()
	var current checkCase
	steps := []struct {
		path       string
		body       func() any
		wantStatus int
		decodeCase bool
	}{
		{"/api/v1/cases", func() any {
			return map[string]any{
				"id": "selfcheck-case", "idempotencyKey": "create-1", "actor": "档案整理员",
				"title": "自检口述史", "collectionUnit": "地方档案馆", "intervieweeCode": "INT-001", "sourceRef": "archive://selfcheck/audio",
				"requestedScope": map[string]any{"audiences": []string{"公众"}, "purposes": []string{"学术研究"}},
			}
		}, http.StatusCreated, true},
		{"/api/v1/cases/selfcheck-case/consents", func() any {
			return map[string]any{
				"expectedRevision": current.Revision, "idempotencyKey": "consent-1", "actor": "档案整理员", "id": "consent-selfcheck",
				"evidenceRef": "consent://signed/selfcheck", "allowedAudiences": []string{"公众"}, "allowedPurposes": []string{"学术研究"},
				"effectiveAt": now.Add(-time.Hour), "expiresAt": now.Add(24 * time.Hour),
			}
		}, http.StatusOK, true},
		{"/api/v1/cases/selfcheck-case/transcripts", func() any {
			return map[string]any{
				"expectedRevision": current.Revision, "idempotencyKey": "transcript-1", "actor": "档案整理员", "id": "transcript-v1", "baseVersionId": "",
				"segments": []map[string]any{{"id": "seg-1", "startMs": 0, "endMs": 5000, "text": "讲述传统技艺与家庭传承。"}},
			}
		}, http.StatusOK, true},
		{"/api/v1/cases/selfcheck-case/findings", func() any {
			return map[string]any{
				"expectedRevision": current.Revision, "idempotencyKey": "finding-1", "actor": "档案整理员", "id": "finding-1",
				"transcriptVersionId": "transcript-v1", "segmentId": "seg-1", "category": "个人隐私", "riskReason": "包含亲属信息", "redactionProposal": "以匿名关系替代姓名",
			}
		}, http.StatusOK, true},
		{"/api/v1/cases/selfcheck-case/objections", func() any {
			return map[string]any{
				"expectedRevision": current.Revision, "idempotencyKey": "objection-1", "actor": "伦理复核员", "id": "objection-1", "findingId": "finding-1", "reason": "需要新转写固化匿名化结果",
			}
		}, http.StatusOK, true},
		{"/api/v1/cases/selfcheck-case/transcripts", func() any {
			return map[string]any{
				"expectedRevision": current.Revision, "idempotencyKey": "transcript-2", "actor": "档案整理员", "id": "transcript-v2", "baseVersionId": "transcript-v1",
				"segments": []map[string]any{{"id": "seg-1", "startMs": 0, "endMs": 5000, "text": "讲述传统技艺与一位家庭成员的传承。"}},
			}
		}, http.StatusOK, true},
		{"/api/v1/cases/selfcheck-case/objections/objection-1/close", func() any {
			return map[string]any{
				"expectedRevision": current.Revision, "idempotencyKey": "close-1", "actor": "伦理复核员", "resolutionEvidenceRef": "transcript-v2",
			}
		}, http.StatusOK, true},
		{"/api/v1/cases/selfcheck-case/freeze", func() any {
			preview := map[string]any{}
			request, _ := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/api/v1/cases/selfcheck-case/freeze-preview", nil)
			response, _ := client.Do(request)
			if response != nil {
				defer response.Body.Close()
				_ = json.NewDecoder(response.Body).Decode(&preview)
			}
			return map[string]any{
				"expectedRevision": current.Revision, "idempotencyKey": "freeze-1", "actor": "档案整理员", "manifestId": "manifest-selfcheck", "confirmedManifestDigest": preview["manifestDigest"],
			}
		}, http.StatusOK, false},
	}
	for index, step := range steps {
		if err := postCheck(client, baseURL+step.path, step.body(), step.wantStatus, func(data []byte) error {
			if step.decodeCase {
				return json.Unmarshal(data, &current)
			}
			return nil
		}); err != nil {
			return shutdownAfterCheck(server, serveResult, fmt.Errorf("自检步骤 %d: %w", index+1, err))
		}
		if index == len(steps)-1 {
			current.Revision++
		}
	}
	if err := postCheck(client, baseURL+"/api/v1/cases/selfcheck-case/authorize", map[string]any{
		"expectedRevision": current.Revision, "idempotencyKey": "authorize-1", "actor": "公开负责人", "credentialId": "credential-selfcheck",
		"scope": map[string]any{"audiences": []string{"公众"}, "purposes": []string{"学术研究"}},
	}, http.StatusOK, func(data []byte) error { return nil }); err != nil {
		return shutdownAfterCheck(server, serveResult, err)
	}
	current.Revision++
	if err := postCheck(client, baseURL+"/api/v1/cases/selfcheck-case/transcripts", map[string]any{
		"expectedRevision": int64(1), "idempotencyKey": "stale-check", "actor": "档案整理员", "id": "must-not-exist", "segments": []any{},
	}, http.StatusConflict, nil); err != nil {
		return shutdownAfterCheck(server, serveResult, fmt.Errorf("陈旧版本阻断失败: %w", err))
	}
	verification, err := getVerification(client, baseURL+"/api/v1/cases/selfcheck-case/credentials/credential-selfcheck/verify")
	if err != nil || !verification.Valid {
		return shutdownAfterCheck(server, serveResult, fmt.Errorf("凭据首次验真失败: %v", err))
	}
	if err := postCheck(client, baseURL+"/api/v1/cases/selfcheck-case/consents/consent-selfcheck/withdraw", map[string]any{
		"expectedRevision": current.Revision, "idempotencyKey": "withdraw-1", "actor": "档案整理员",
	}, http.StatusOK, nil); err != nil {
		return shutdownAfterCheck(server, serveResult, err)
	}
	verification, err = getVerification(client, baseURL+"/api/v1/cases/selfcheck-case/credentials/credential-selfcheck/verify")
	if err != nil || verification.Valid {
		return shutdownAfterCheck(server, serveResult, fmt.Errorf("撤回同意后凭据未失效: %v", err))
	}
	fmt.Println("selfcheck 通过：建档、同意、转写、治理、整改、冻结、授权、验真与撤回失效流程均成功")
	return shutdownAfterCheck(server, serveResult, nil)
}

func waitReady(ctx context.Context, client *http.Client, url string) error {
	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()
	for {
		request, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		response, err := client.Do(request)
		if err == nil {
			_ = response.Body.Close()
			if response.StatusCode == http.StatusOK {
				return nil
			}
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("等待服务就绪超时")
		case <-ticker.C:
		}
	}
}

func postCheck(client *http.Client, url string, value any, want int, inspect func([]byte) error) error {
	body, err := json.Marshal(value)
	if err != nil {
		return err
	}
	response, err := client.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer response.Body.Close()
	data, err := io.ReadAll(io.LimitReader(response.Body, 2<<20))
	if err != nil {
		return err
	}
	if response.StatusCode != want {
		return fmt.Errorf("%s 返回 %d，期望 %d：%s", url, response.StatusCode, want, string(data))
	}
	if inspect != nil {
		return inspect(data)
	}
	return nil
}

func getVerification(client *http.Client, url string) (domain.CredentialVerification, error) {
	response, err := client.Get(url)
	if err != nil {
		return domain.CredentialVerification{}, err
	}
	defer response.Body.Close()
	var result domain.CredentialVerification
	if response.StatusCode != http.StatusOK {
		return result, fmt.Errorf("验真接口返回 %d", response.StatusCode)
	}
	err = json.NewDecoder(response.Body).Decode(&result)
	return result, err
}

func shutdownAfterCheck(server *http.Server, serveResult <-chan error, checkErr error) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	shutdownErr := server.Shutdown(ctx)
	select {
	case <-serveResult:
	case <-ctx.Done():
		if checkErr == nil {
			checkErr = fmt.Errorf("服务关闭超时")
		}
	}
	if checkErr != nil {
		return checkErr
	}
	return shutdownErr
}
