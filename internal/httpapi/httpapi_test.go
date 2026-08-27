package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/benzhi/oral-history-release/internal/application"
	"github.com/benzhi/oral-history-release/internal/store"
)

func testAPI(t *testing.T) http.Handler {
	t.Helper()
	repository, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return New(application.NewService(repository)).Handler()
}

func TestWorkbenchAndCaseAPI(t *testing.T) {
	handler := testAPI(t)
	page := httptest.NewRecorder()
	handler.ServeHTTP(page, httptest.NewRequest(http.MethodGet, "/", nil))
	if page.Code != http.StatusOK || !strings.Contains(page.Body.String(), "口述史资料公开授权工作台") {
		t.Fatalf("workbench unavailable: %d", page.Code)
	}
	body := map[string]any{"id": "web-case", "idempotencyKey": "k1", "actor": "整理员", "title": "资料", "collectionUnit": "档案馆", "intervieweeCode": "P1", "sourceRef": "source", "requestedScope": map[string]any{"audiences": []string{"公众"}, "purposes": []string{"研究"}}}
	data, _ := json.Marshal(body)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/cases", bytes.NewReader(data))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("create returned %d: %s", response.Code, response.Body.String())
	}
	detail := httptest.NewRecorder()
	handler.ServeHTTP(detail, httptest.NewRequest(http.MethodGet, "/api/v1/cases/web-case", nil))
	if detail.Code != http.StatusOK || !strings.Contains(detail.Body.String(), "blockingItems") {
		t.Fatalf("detail unavailable: %s", detail.Body.String())
	}
}

func TestJSONProtocolGuards(t *testing.T) {
	handler := testAPI(t)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/cases", strings.NewReader("{}"))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("want 415, got %d", response.Code)
	}
	request = httptest.NewRequest(http.MethodPost, "/api/v1/cases", strings.NewReader(`{"unknown":true}`))
	request.Header.Set("Content-Type", "application/json")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("want strict json 400, got %d", response.Code)
	}
}
