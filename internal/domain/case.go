package domain

import (
	"strings"
	"time"
)

type CreateCaseInput struct {
	ID              string
	Title           string
	CollectionUnit  string
	IntervieweeCode string
	SourceRef       string
	RequestedScope  PublicationScope
	Now             time.Time
}

func NewCase(in CreateCaseInput) (*OralHistoryCase, error) {
	if strings.TrimSpace(in.ID) == "" {
		return nil, Invalid("id", "不能为空")
	}
	if strings.TrimSpace(in.Title) == "" {
		return nil, Invalid("title", "不能为空")
	}
	if strings.TrimSpace(in.CollectionUnit) == "" {
		return nil, Invalid("collectionUnit", "不能为空")
	}
	if strings.TrimSpace(in.IntervieweeCode) == "" {
		return nil, Invalid("intervieweeCode", "不能为空")
	}
	if strings.TrimSpace(in.SourceRef) == "" {
		return nil, Invalid("sourceRef", "不能为空")
	}
	if err := validateScope(in.RequestedScope); err != nil {
		return nil, err
	}
	now := in.Now.UTC()
	return &OralHistoryCase{
		ID: in.ID, Title: strings.TrimSpace(in.Title), CollectionUnit: strings.TrimSpace(in.CollectionUnit),
		IntervieweeCode: strings.TrimSpace(in.IntervieweeCode), SourceRef: strings.TrimSpace(in.SourceRef),
		State: StateDraft, Revision: 1, CreatedAt: now, UpdatedAt: now,
		RequestedScope: normalizeScope(in.RequestedScope),
	}, nil
}

func (c *OralHistoryCase) Mutable() error {
	if c.State == StateFrozen || c.State == StateAuthorized {
		return ErrFrozen
	}
	return nil
}

func (c *OralHistoryCase) Touch(now time.Time) {
	c.Revision++
	c.UpdatedAt = now.UTC()
}

func (c *OralHistoryCase) LatestTranscript() *TranscriptVersion {
	if len(c.Transcripts) == 0 {
		return nil
	}
	return &c.Transcripts[len(c.Transcripts)-1]
}

func (c *OralHistoryCase) findTranscript(id string) *TranscriptVersion {
	for i := range c.Transcripts {
		if c.Transcripts[i].ID == id {
			return &c.Transcripts[i]
		}
	}
	return nil
}

func (c *OralHistoryCase) findFinding(id string) *SensitiveFinding {
	for i := range c.Findings {
		if c.Findings[i].ID == id {
			return &c.Findings[i]
		}
	}
	return nil
}

func (c *OralHistoryCase) RecalculateState(now time.Time) {
	if c.State == StateFrozen || c.State == StateAuthorized {
		return
	}
	if !c.ConsentCovers(c.RequestedScope, now) {
		c.State = StateDraft
		return
	}
	if len(c.Transcripts) == 0 {
		c.State = StateConsentVerified
		return
	}
	for _, o := range c.Objections {
		if o.ClosedAt == nil {
			c.State = StateRemediation
			return
		}
	}
	for _, f := range c.Findings {
		if f.Status != FindingResolved {
			c.State = StateGoverning
			return
		}
	}
	c.State = StateFreezable
}
