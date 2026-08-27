package httpapi

import (
	"encoding/json"
	"errors"
	"mime"
	"net/http"
	"strings"

	"github.com/benzhi/oral-history-release/internal/domain"
)

const maxRequestBody = 1 << 20

type errorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeRawJSON(w http.ResponseWriter, status int, body json.RawMessage) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write(body)
	_, _ = w.Write([]byte("\n"))
}

func writeError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	code := "internal_error"
	message := "服务内部错误"
	var domainError *domain.Error
	if errors.As(err, &domainError) {
		code, message = domainError.Code, domainError.Message
		switch code {
		case "not_found":
			status = http.StatusNotFound
		case "revision_conflict", "case_exists", "manifest_conflict":
			status = http.StatusConflict
		case "unsupported_media_type":
			status = http.StatusUnsupportedMediaType
		case "case_frozen", "consent_blocked", "freeze_blocked", "not_frozen", "scope_exceeded", "open_objection", "unresolved_finding", "findings_batch_too_large":
			status = http.StatusUnprocessableEntity
		default:
			status = http.StatusBadRequest
		}
	}
	writeJSON(w, status, errorBody{Code: code, Message: message})
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) error {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		return domain.NewError("unsupported_media_type", "写请求必须使用 application/json")
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBody)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return domain.NewError("invalid_json", "JSON 请求体不合法："+err.Error())
	}
	var extra any
	if err := decoder.Decode(&extra); err == nil {
		return domain.NewError("invalid_json", "请求体只能包含一个 JSON 对象")
	}
	return nil
}

func requestID(r *http.Request, name string) (string, error) {
	value := strings.TrimSpace(r.PathValue(name))
	if value == "" || strings.ContainsAny(value, `/\\`) {
		return "", domain.Invalid(name, "路径标识不合法")
	}
	return value, nil
}
