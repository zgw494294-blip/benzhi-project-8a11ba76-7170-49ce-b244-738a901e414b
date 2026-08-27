package domain

import (
	"crypto/subtle"
	"strings"
	"time"
)

type credentialCanonical struct {
	ID                   string   `json:"id"`
	CaseID               string   `json:"caseId"`
	FrozenManifestDigest string   `json:"frozenManifestDigest"`
	AudienceScope        []string `json:"audienceScope"`
	PurposeScope         []string `json:"purposeScope"`
	IssuedBy             string   `json:"issuedBy"`
	IssuedAt             string   `json:"issuedAt"`
}

func credentialDigest(credential ReleaseCredential) string {
	return Digest(credentialCanonical{
		ID: credential.ID, CaseID: credential.CaseID,
		FrozenManifestDigest: credential.FrozenManifestDigest,
		AudienceScope:        normalizeStrings(credential.AudienceScope), PurposeScope: normalizeStrings(credential.PurposeScope),
		IssuedBy: credential.IssuedBy, IssuedAt: credential.IssuedAt.UTC().Format(time.RFC3339Nano),
	})
}

func (c *OralHistoryCase) Authorize(id string, scope PublicationScope, actor string, now time.Time) (*ReleaseCredential, error) {
	if c.State != StateFrozen || c.FrozenManifest == nil {
		return nil, NewError("not_frozen", "只有已冻结案卷可以签发授权")
	}
	if strings.TrimSpace(id) == "" || strings.TrimSpace(actor) == "" {
		return nil, Invalid("credential", "id 与 actor 不能为空")
	}
	if err := validateScope(scope); err != nil {
		return nil, err
	}
	scope = normalizeScope(scope)
	if !containsAll(c.RequestedScope.Audiences, scope.Audiences) || !containsAll(c.RequestedScope.Purposes, scope.Purposes) {
		return nil, NewError("scope_exceeded", "签发边界超出案卷拟公开范围")
	}
	if !c.ConsentCovers(scope, now) {
		return nil, ErrConsentBlocked
	}
	credential := ReleaseCredential{
		ID: id, CaseID: c.ID, FrozenManifestDigest: c.FrozenManifest.ManifestDigest,
		AudienceScope: scope.Audiences, PurposeScope: scope.Purposes,
		IssuedBy: actor, IssuedAt: now.UTC(), Status: CredentialActive,
	}
	credential.VerificationDigest = credentialDigest(credential)
	c.Credentials = append(c.Credentials, credential)
	c.State = StateAuthorized
	c.Touch(now)
	return &c.Credentials[len(c.Credentials)-1], nil
}

type CredentialVerification struct {
	CredentialID string `json:"credentialId"`
	Valid        bool   `json:"valid"`
	Status       string `json:"status"`
	Reason       string `json:"reason"`
	Digest       string `json:"digest"`
}

func (c *OralHistoryCase) VerifyCredential(id string, now time.Time) CredentialVerification {
	result := CredentialVerification{CredentialID: id, Status: string(CredentialInvalid)}
	for _, credential := range c.Credentials {
		if credential.ID != id {
			continue
		}
		result.Digest = credential.VerificationDigest
		computed := credentialDigest(credential)
		if subtle.ConstantTimeCompare([]byte(computed), []byte(credential.VerificationDigest)) != 1 {
			result.Reason = "凭据校验摘要不一致"
			return result
		}
		if c.FrozenManifest == nil || RecalculateManifestDigest(*c.FrozenManifest) != credential.FrozenManifestDigest {
			result.Reason = "冻结摘要不存在或已损坏"
			return result
		}
		scope := PublicationScope{Audiences: credential.AudienceScope, Purposes: credential.PurposeScope}
		if !c.ConsentCovers(scope, now) {
			result.Reason = "知情同意已撤回、过期或不再覆盖公开边界"
			return result
		}
		result.Valid = true
		result.Status = string(CredentialActive)
		result.Reason = "凭据摘要、冻结证据和当前同意状态均有效"
		return result
	}
	result.Reason = "凭据不存在"
	return result
}
