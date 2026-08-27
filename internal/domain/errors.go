package domain

import "fmt"

// Error 是可稳定映射到 HTTP 响应的领域错误。
type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *Error) Error() string { return e.Code + ": " + e.Message }

func NewError(code, message string) *Error { return &Error{Code: code, Message: message} }

func Invalid(field, reason string) *Error {
	return NewError("invalid_"+field, fmt.Sprintf("%s不合法：%s", field, reason))
}

var (
	ErrNotFound          = NewError("not_found", "案卷或证据不存在")
	ErrRevisionConflict  = NewError("revision_conflict", "expectedRevision 与当前案卷版本不一致")
	ErrConsentBlocked    = NewError("consent_blocked", "知情同意未完整覆盖拟公开范围")
	ErrFrozen            = NewError("case_frozen", "案卷冻结后禁止修改历史证据")
	ErrOpenObjection     = NewError("open_objection", "仍有未闭环异议")
	ErrUnresolvedFinding = NewError("unresolved_finding", "仍有未处置敏感片段")
)

func ErrorCode(err error) string {
	if e, ok := err.(*Error); ok {
		return e.Code
	}
	return "internal_error"
}
