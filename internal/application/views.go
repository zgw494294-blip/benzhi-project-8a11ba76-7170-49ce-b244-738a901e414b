package application

import (
	"time"

	"github.com/benzhi/oral-history-release/internal/domain"
)

type CaseSummary struct {
	ID              string           `json:"id"`
	Title           string           `json:"title"`
	IntervieweeCode string           `json:"intervieweeCode"`
	State           domain.CaseState `json:"state"`
	Revision        int64            `json:"revision"`
	UpdatedAt       time.Time        `json:"updatedAt"`
}

type CaseDetail struct {
	Case                 *domain.OralHistoryCase        `json:"case"`
	BlockingItems        []domain.BlockingItem          `json:"blockingItems"`
	GovernanceConclusion string                         `json:"governanceConclusion"`
	LatestDifferences    []domain.SegmentDifference     `json:"latestDifferences"`
	ConsentCoverage      domain.ConsentCoverage         `json:"consentCoverage"`
	TranscriptImpact     domain.TranscriptImpactSummary `json:"transcriptImpact"`
	FreezePreview        domain.FreezePreview           `json:"freezePreview"`
	AuditTimeline        []domain.AuditEvent            `json:"auditTimeline"`
}

func (s *Service) GetCase(caseID string) (*CaseDetail, error) {
	caseFile, audits, err := s.repository.Get(caseID)
	if err != nil {
		return nil, err
	}
	var base, current *domain.TranscriptVersion
	if count := len(caseFile.Transcripts); count > 0 {
		current = &caseFile.Transcripts[count-1]
		if count > 1 {
			base = &caseFile.Transcripts[count-2]
		}
	}
	now := s.now()
	caseFile.RecalculateState(now)
	diffs := domain.DiffTranscripts(base, current)
	for i := range diffs {
		for _, finding := range caseFile.Findings {
			if base != nil && finding.TranscriptVersionID == base.ID && finding.SegmentID == diffs[i].SegmentID {
				diffs[i].FindingIDs = append(diffs[i].FindingIDs, finding.ID)
			}
		}
	}
	impact := domain.TranscriptImpactSummary{}
	for _, diff := range diffs {
		switch diff.Kind {
		case "新增":
			impact.Added++
		case "删除":
			impact.Deleted++
		case "改文":
			impact.TextChanged++
		case "改时":
			impact.TimeChanged++
		case "改文改时":
			impact.TextChanged++
			impact.TimeChanged++
		}
		impact.AffectedFindings += len(diff.FindingIDs)
	}
	return &CaseDetail{
		Case: caseFile, BlockingItems: caseFile.BlockingItems(now),
		GovernanceConclusion: caseFile.GovernanceConclusion(now),
		LatestDifferences:    diffs, ConsentCoverage: caseFile.ConsentCoverage(caseFile.RequestedScope, now), TranscriptImpact: impact,
		FreezePreview: caseFile.FreezePreview(now), AuditTimeline: audits,
	}, nil
}

func (s *Service) GetFreezePreview(caseID string) (domain.FreezePreview, error) {
	caseFile, _, err := s.repository.Get(caseID)
	if err != nil {
		return domain.FreezePreview{}, err
	}
	return caseFile.FreezePreview(s.now()), nil
}

func (s *Service) ListCases() ([]CaseSummary, error) {
	cases, err := s.repository.List()
	if err != nil {
		return nil, err
	}
	result := make([]CaseSummary, 0, len(cases))
	now := s.now()
	for _, caseFile := range cases {
		caseFile.RecalculateState(now)
		result = append(result, CaseSummary{
			ID: caseFile.ID, Title: caseFile.Title, IntervieweeCode: caseFile.IntervieweeCode,
			State: caseFile.State, Revision: caseFile.Revision, UpdatedAt: caseFile.UpdatedAt,
		})
	}
	return result, nil
}
