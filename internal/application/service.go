package application

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/benzhi/oral-history-release/internal/domain"
	"github.com/benzhi/oral-history-release/internal/store"
)

type Service struct {
	repository        *store.Repository
	now               func() time.Time
	verificationMu    sync.RWMutex
	verificationCache map[string]cachedCredentialVerification
}

func NewService(repository *store.Repository) *Service {
	return &Service{
		repository:        repository,
		now:               time.Now,
		verificationCache: make(map[string]cachedCredentialVerification),
	}
}

func (s *Service) execute(caseID, action string, meta CommandMeta, mutation store.Mutation) (json.RawMessage, error) {
	if err := meta.Validate(); err != nil {
		return nil, err
	}
	result := s.repository.Execute(caseID, meta.ExpectedRevision, meta.IdempotencyKey, action, meta.Actor, s.now(), mutation)
	return result.Body, result.Err
}

func (s *Service) CreateCase(command CreateCaseCommand) (json.RawMessage, error) {
	if err := command.CreateMeta.Validate(); err != nil {
		return nil, err
	}
	caseID := command.ID
	if caseID == "" {
		caseID = stableID("case", command.Actor+"\x00"+command.IdempotencyKey)
	}
	now := s.now()
	caseFile, err := domain.NewCase(domain.CreateCaseInput{
		ID: caseID, Title: command.Title, CollectionUnit: command.CollectionUnit,
		IntervieweeCode: command.IntervieweeCode, SourceRef: command.SourceRef,
		RequestedScope: command.RequestedScope, Now: now,
	})
	if err != nil {
		return nil, err
	}
	result := s.repository.Create(caseFile, command.IdempotencyKey, command.Actor, now)
	return result.Body, result.Err
}

func (s *Service) AddConsent(caseID string, command AddConsentCommand) (json.RawMessage, error) {
	if command.ID == "" {
		id, err := newID("consent")
		if err != nil {
			return nil, err
		}
		command.ID = id
	}
	return s.execute(caseID, "add_consent", command.CommandMeta, func(c *domain.OralHistoryCase) (any, error) {
		grant := domain.ConsentGrant{
			ID: command.ID, EvidenceRef: command.EvidenceRef,
			AllowedAudiences: command.AllowedAudiences, AllowedPurposes: command.AllowedPurposes,
			EffectiveAt: command.EffectiveAt, ExpiresAt: command.ExpiresAt,
		}
		if err := c.AddConsent(grant, s.now()); err != nil {
			return nil, err
		}
		return c, nil
	})
}

func (s *Service) SubmitTranscript(caseID string, command SubmitTranscriptCommand) (json.RawMessage, error) {
	if command.ID == "" {
		id, err := newID("transcript")
		if err != nil {
			return nil, err
		}
		command.ID = id
	}
	return s.execute(caseID, "submit_transcript", command.CommandMeta, func(c *domain.OralHistoryCase) (any, error) {
		version := domain.TranscriptVersion{
			ID: command.ID, BaseVersionID: command.BaseVersionID,
			Segments: command.Segments, SubmittedBy: command.Actor,
		}
		if err := c.SubmitTranscript(version, s.now()); err != nil {
			return nil, err
		}
		return c, nil
	})
}

func (s *Service) AddFinding(caseID string, command AddFindingCommand) (json.RawMessage, error) {
	if command.ID == "" {
		id, err := newID("finding")
		if err != nil {
			return nil, err
		}
		command.ID = id
	}
	return s.execute(caseID, "add_finding", command.CommandMeta, func(c *domain.OralHistoryCase) (any, error) {
		finding := domain.SensitiveFinding{
			ID: command.ID, TranscriptVersionID: command.TranscriptVersionID,
			SegmentID: command.SegmentID, Category: command.Category,
			RiskReason: command.RiskReason, RedactionProposal: command.RedactionProposal,
		}
		if err := c.AddFinding(finding, s.now()); err != nil {
			return nil, err
		}
		return c, nil
	})
}

func (s *Service) AddFindingsBatch(caseID string, command BatchFindingsCommand) (json.RawMessage, error) {
	items := command.Items
	if len(items) == 0 {
		items = command.Findings
	}
	if len(items) == 0 {
		return nil, domain.Invalid("findings", "至少包含一条敏感项")
	}
	if err := command.CommandMeta.Validate(); err != nil {
		return nil, err
	}
	return s.execute(caseID, "add_findings_batch", command.CommandMeta, func(c *domain.OralHistoryCase) (any, error) {
		findings := make([]domain.SensitiveFinding, 0, len(items))
		for i, item := range items {
			id := item.ID
			if id == "" {
				id = stableID("finding", command.IdempotencyKey+":"+string(rune(i)))
			}
			findings = append(findings, domain.SensitiveFinding{ID: id, TranscriptVersionID: item.TranscriptVersionID, SegmentID: item.SegmentID, Category: item.Category, RiskReason: item.RiskReason, RedactionProposal: item.RedactionProposal})
		}
		return c.AddFindingsBatch(findings, s.now())
	})
}

func (s *Service) ReviseFinding(caseID, findingID string, command ReviseFindingCommand) (json.RawMessage, error) {
	return s.execute(caseID, "revise_finding", command.CommandMeta, func(c *domain.OralHistoryCase) (any, error) {
		if err := c.ReviseFinding(findingID, command.RedactionProposal, command.ResolutionEvidenceRef, s.now()); err != nil {
			return nil, err
		}
		return c, nil
	})
}

func (s *Service) RaiseObjection(caseID string, command RaiseObjectionCommand) (json.RawMessage, error) {
	if command.ID == "" {
		id, err := newID("objection")
		if err != nil {
			return nil, err
		}
		command.ID = id
	}
	return s.execute(caseID, "raise_objection", command.CommandMeta, func(c *domain.OralHistoryCase) (any, error) {
		objection := domain.Objection{ID: command.ID, Reason: command.Reason, FindingID: command.FindingID, RaisedBy: command.Actor}
		if err := c.RaiseObjection(objection, s.now()); err != nil {
			return nil, err
		}
		return c, nil
	})
}

func (s *Service) CloseObjection(caseID, objectionID string, command CloseObjectionCommand) (json.RawMessage, error) {
	return s.execute(caseID, "close_objection", command.CommandMeta, func(c *domain.OralHistoryCase) (any, error) {
		if err := c.CloseObjection(objectionID, command.ResolutionEvidenceRef, command.Actor, s.now()); err != nil {
			return nil, err
		}
		return c, nil
	})
}
