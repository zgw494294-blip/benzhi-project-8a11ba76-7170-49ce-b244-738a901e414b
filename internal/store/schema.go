package store

import (
	"encoding/json"

	"github.com/benzhi/oral-history-release/internal/domain"
)

const schemaVersion = 1

type storedResult struct {
	Body         []byte `json:"body,omitempty"`
	ErrorCode    string `json:"errorCode,omitempty"`
	ErrorMessage string `json:"errorMessage,omitempty"`
}

type caseRecord struct {
	SchemaVersion int                     `json:"schemaVersion"`
	Aggregate     domain.OralHistoryCase  `json:"aggregate"`
	Audits        []domain.AuditEvent     `json:"audits"`
	Idempotency   map[string]storedResult `json:"idempotency"`
	ObjectDigests []string                `json:"objectDigests"`
}

type objectEnvelope struct {
	SchemaVersion int             `json:"schemaVersion"`
	Kind          string          `json:"kind"`
	Digest        string          `json:"digest"`
	Payload       json.RawMessage `json:"payload"`
}

type Result struct {
	Body      json.RawMessage
	Err       error
	FromCache bool
}
