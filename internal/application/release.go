package application

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/benzhi/oral-history-release/internal/domain"
)

func (s *Service) Freeze(caseID string, command FreezeCommand) (json.RawMessage, error) {
	if strings.TrimSpace(command.ConfirmedManifestDigest) == "" {
		return nil, domain.Invalid("confirmedManifestDigest", "必须确认冻结候选摘要")
	}
	if command.ManifestID == "" {
		id, err := newID("manifest")
		if err != nil {
			return nil, err
		}
		command.ManifestID = id
	}
	return s.execute(caseID, "freeze", command.CommandMeta, func(c *domain.OralHistoryCase) (any, error) {
		manifest, err := c.FreezeConfirmed(command.ManifestID, command.Actor, command.ConfirmedManifestDigest, s.now())
		if err != nil {
			return nil, err
		}
		return manifest, nil
	})
}

func (s *Service) Authorize(caseID string, command AuthorizeCommand) (json.RawMessage, error) {
	if command.CredentialID == "" {
		id, err := newID("credential")
		if err != nil {
			return nil, err
		}
		command.CredentialID = id
	}
	return s.execute(caseID, "authorize", command.CommandMeta, func(c *domain.OralHistoryCase) (any, error) {
		credential, err := c.Authorize(command.CredentialID, command.Scope, command.Actor, s.now())
		if err != nil {
			return nil, err
		}
		return credential, nil
	})
}

func (s *Service) WithdrawConsent(caseID, consentID string, command WithdrawConsentCommand) (json.RawMessage, error) {
	return s.WithdrawConsentContext(context.Background(), caseID, consentID, command)
}

func (s *Service) WithdrawConsentContext(ctx context.Context, caseID, consentID string, command WithdrawConsentCommand) (json.RawMessage, error) {
	return s.executeContext(ctx, caseID, "withdraw_consent", command.CommandMeta, func(c *domain.OralHistoryCase) (any, error) {
		if err := c.WithdrawConsent(consentID, s.now()); err != nil {
			return nil, err
		}
		return c, nil
	})
}

func (s *Service) VerifyCredential(caseID, credentialID string) (domain.CredentialVerification, error) {
	caseFile, _, err := s.repository.Get(caseID)
	if err != nil {
		return domain.CredentialVerification{}, err
	}
	return caseFile.VerifyCredential(credentialID, s.now()), nil
}
