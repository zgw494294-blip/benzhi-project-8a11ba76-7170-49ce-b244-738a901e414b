package application

import (
	"time"

	"github.com/benzhi/oral-history-release/internal/domain"
)

type CreateCaseCommand struct {
	CreateMeta
	ID              string                  `json:"id,omitempty"`
	Title           string                  `json:"title"`
	CollectionUnit  string                  `json:"collectionUnit"`
	IntervieweeCode string                  `json:"intervieweeCode"`
	SourceRef       string                  `json:"sourceRef"`
	RequestedScope  domain.PublicationScope `json:"requestedScope"`
}

type AddConsentCommand struct {
	CommandMeta
	ID               string    `json:"id,omitempty"`
	EvidenceRef      string    `json:"evidenceRef"`
	AllowedAudiences []string  `json:"allowedAudiences"`
	AllowedPurposes  []string  `json:"allowedPurposes"`
	EffectiveAt      time.Time `json:"effectiveAt"`
	ExpiresAt        time.Time `json:"expiresAt"`
}

type SubmitTranscriptCommand struct {
	CommandMeta
	ID            string                     `json:"id,omitempty"`
	BaseVersionID string                     `json:"baseVersionId,omitempty"`
	Segments      []domain.TranscriptSegment `json:"segments"`
}

type AddFindingCommand struct {
	CommandMeta
	ID                  string              `json:"id,omitempty"`
	TranscriptVersionID string              `json:"transcriptVersionId"`
	SegmentID           string              `json:"segmentId"`
	Category            string              `json:"category"`
	RiskReason          string              `json:"riskReason"`
	RedactionProposal   string              `json:"redactionProposal"`
	Items               []AddFindingCommand `json:"items,omitempty"`
	Findings            []AddFindingCommand `json:"findings,omitempty"`
}

type BatchFindingsCommand struct {
	CommandMeta
	Items    []AddFindingCommand `json:"items,omitempty"`
	Findings []AddFindingCommand `json:"findings,omitempty"`
}

type ReviseFindingCommand struct {
	CommandMeta
	RedactionProposal     string `json:"redactionProposal"`
	ResolutionEvidenceRef string `json:"resolutionEvidenceRef"`
}

type RaiseObjectionCommand struct {
	CommandMeta
	ID        string `json:"id,omitempty"`
	Reason    string `json:"reason"`
	FindingID string `json:"findingId,omitempty"`
}

type CloseObjectionCommand struct {
	CommandMeta
	ResolutionEvidenceRef string `json:"resolutionEvidenceRef"`
}

type FreezeCommand struct {
	CommandMeta
	ManifestID              string `json:"manifestId,omitempty"`
	ConfirmedManifestDigest string `json:"confirmedManifestDigest"`
}

type AuthorizeCommand struct {
	CommandMeta
	CredentialID string                  `json:"credentialId,omitempty"`
	Scope        domain.PublicationScope `json:"scope"`
}

type WithdrawConsentCommand struct {
	CommandMeta
}
