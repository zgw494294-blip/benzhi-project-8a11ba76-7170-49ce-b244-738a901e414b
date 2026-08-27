package application

import (
	"strings"

	"github.com/benzhi/oral-history-release/internal/domain"
)

type CommandMeta struct {
	ExpectedRevision int64  `json:"expectedRevision"`
	IdempotencyKey   string `json:"idempotencyKey"`
	Actor            string `json:"actor"`
}

func (m CommandMeta) Validate() error {
	if m.ExpectedRevision < 1 {
		return domain.Invalid("expectedRevision", "必须大于零")
	}
	if strings.TrimSpace(m.IdempotencyKey) == "" {
		return domain.Invalid("idempotencyKey", "不能为空")
	}
	if len(m.IdempotencyKey) > 128 {
		return domain.Invalid("idempotencyKey", "长度不得超过 128")
	}
	if strings.TrimSpace(m.Actor) == "" {
		return domain.Invalid("actor", "不能为空")
	}
	return nil
}

type CreateMeta struct {
	IdempotencyKey string `json:"idempotencyKey"`
	Actor          string `json:"actor"`
}

func (m CreateMeta) Validate() error {
	if strings.TrimSpace(m.IdempotencyKey) == "" || strings.TrimSpace(m.Actor) == "" {
		return domain.Invalid("metadata", "idempotencyKey 与 actor 不能为空")
	}
	return nil
}
