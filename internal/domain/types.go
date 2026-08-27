package domain

import "time"

type CaseState string

const (
	StateDraft           CaseState = "草稿"
	StateConsentVerified CaseState = "同意已核验"
	StateGoverning       CaseState = "治理中"
	StateRemediation     CaseState = "待整改"
	StateFreezable       CaseState = "可冻结"
	StateFrozen          CaseState = "已冻结"
	StateAuthorized      CaseState = "已授权"
)

const RuleVersion = "oral-history-governance/1.0"

type OralHistoryCase struct {
	ID              string              `json:"id"`
	Title           string              `json:"title"`
	CollectionUnit  string              `json:"collectionUnit"`
	IntervieweeCode string              `json:"intervieweeCode"`
	SourceRef       string              `json:"sourceRef"`
	State           CaseState           `json:"state"`
	Revision        int64               `json:"revision"`
	CreatedAt       time.Time           `json:"createdAt"`
	UpdatedAt       time.Time           `json:"updatedAt"`
	RequestedScope  PublicationScope    `json:"requestedScope"`
	Consents        []ConsentGrant      `json:"consents"`
	Transcripts     []TranscriptVersion `json:"transcripts"`
	Findings        []SensitiveFinding  `json:"findings"`
	Objections      []Objection         `json:"objections"`
	FrozenManifest  *FrozenManifest     `json:"frozenManifest,omitempty"`
	Credentials     []ReleaseCredential `json:"credentials"`
}

type PublicationScope struct {
	Audiences []string `json:"audiences"`
	Purposes  []string `json:"purposes"`
}

type ConsentGrant struct {
	ID               string     `json:"id"`
	CaseID           string     `json:"caseId"`
	EvidenceRef      string     `json:"evidenceRef"`
	AllowedAudiences []string   `json:"allowedAudiences"`
	AllowedPurposes  []string   `json:"allowedPurposes"`
	EffectiveAt      time.Time  `json:"effectiveAt"`
	ExpiresAt        time.Time  `json:"expiresAt"`
	WithdrawnAt      *time.Time `json:"withdrawnAt,omitempty"`
	Revision         int64      `json:"revision"`
}

type ConsentCoverageItem struct {
	Dimension   string   `json:"dimension"`
	Value       string   `json:"value"`
	Covered     bool     `json:"covered"`
	ReasonCode  string   `json:"reasonCode"`
	EvidenceRef []string `json:"evidenceRef,omitempty"`
}

type ConsentCoverage struct {
	Items             []ConsentCoverageItem `json:"items"`
	Covered           bool                  `json:"covered"`
	Missing           []ConsentCoverageItem `json:"missing,omitempty"`
	EarliestExpiresAt *time.Time            `json:"earliestExpiresAt,omitempty"`
	RemainingDays     int                   `json:"remainingDays,omitempty"`
	Warning           string                `json:"warning,omitempty"`
}

type TranscriptSegment struct {
	ID      string `json:"id"`
	StartMS int64  `json:"startMs"`
	EndMS   int64  `json:"endMs"`
	Text    string `json:"text"`
}

type TranscriptVersion struct {
	ID            string              `json:"id"`
	CaseID        string              `json:"caseId"`
	BaseVersionID string              `json:"baseVersionId,omitempty"`
	Segments      []TranscriptSegment `json:"segments"`
	SubmittedBy   string              `json:"submittedBy"`
	SubmittedAt   time.Time           `json:"submittedAt"`
	ContentDigest string              `json:"contentDigest"`
}

type TranscriptImpactSummary struct {
	Added            int `json:"added"`
	Deleted          int `json:"deleted"`
	TextChanged      int `json:"textChanged"`
	TimeChanged      int `json:"timeChanged"`
	AffectedFindings int `json:"affectedFindings"`
}

type FindingStatus string

const (
	FindingOpen     FindingStatus = "未处置"
	FindingResolved FindingStatus = "已处置"
)

type SensitiveFinding struct {
	ID                         string        `json:"id"`
	CaseID                     string        `json:"caseId"`
	TranscriptVersionID        string        `json:"transcriptVersionId"`
	SegmentID                  string        `json:"segmentId"`
	Category                   string        `json:"category"`
	RiskReason                 string        `json:"riskReason"`
	RedactionProposal          string        `json:"redactionProposal"`
	Status                     FindingStatus `json:"status"`
	ResolutionEvidenceRef      string        `json:"resolutionEvidenceRef,omitempty"`
	PriorResolutionEvidenceRef string        `json:"priorResolutionEvidenceRef,omitempty"`
}

type Objection struct {
	ID                    string     `json:"id"`
	Reason                string     `json:"reason"`
	FindingID             string     `json:"findingId,omitempty"`
	RaisedBy              string     `json:"raisedBy"`
	RaisedAt              time.Time  `json:"raisedAt"`
	ClosedAt              *time.Time `json:"closedAt,omitempty"`
	ClosedBy              string     `json:"closedBy,omitempty"`
	ResolutionEvidenceRef string     `json:"resolutionEvidenceRef,omitempty"`
}

type FrozenManifest struct {
	ID                  string    `json:"id"`
	CaseID              string    `json:"caseId"`
	TranscriptVersionID string    `json:"transcriptVersionId"`
	EvidenceDigests     []string  `json:"evidenceDigests"`
	ContentDigest       string    `json:"contentDigest"`
	RuleVersion         string    `json:"ruleVersion"`
	ManifestDigest      string    `json:"manifestDigest"`
	FrozenBy            string    `json:"frozenBy"`
	FrozenAt            time.Time `json:"frozenAt"`
}

type ManifestEvidence struct {
	Kind    string `json:"kind"`
	ID      string `json:"id"`
	Digest  string `json:"digest"`
	Summary string `json:"summary"`
}

type FreezePreview struct {
	TranscriptVersionID string             `json:"transcriptVersionId,omitempty"`
	ContentDigest       string             `json:"contentDigest,omitempty"`
	RuleVersion         string             `json:"ruleVersion"`
	Evidence            []ManifestEvidence `json:"evidence"`
	ManifestDigest      string             `json:"manifestDigest,omitempty"`
	BlockingItems       []BlockingItem     `json:"blockingItems,omitempty"`
}

type CredentialStatus string

const (
	CredentialActive  CredentialStatus = "有效"
	CredentialInvalid CredentialStatus = "失效"
)

type ReleaseCredential struct {
	ID                   string           `json:"id"`
	CaseID               string           `json:"caseId"`
	FrozenManifestDigest string           `json:"frozenManifestDigest"`
	AudienceScope        []string         `json:"audienceScope"`
	PurposeScope         []string         `json:"purposeScope"`
	IssuedBy             string           `json:"issuedBy"`
	IssuedAt             time.Time        `json:"issuedAt"`
	VerificationDigest   string           `json:"verificationDigest"`
	Status               CredentialStatus `json:"status"`
}

type AuditEvent struct {
	Sequence       int64     `json:"sequence"`
	CaseID         string    `json:"caseId"`
	Action         string    `json:"action"`
	Actor          string    `json:"actor"`
	OccurredAt     time.Time `json:"occurredAt"`
	PreviousDigest string    `json:"previousDigest"`
	PayloadDigest  string    `json:"payloadDigest"`
	Outcome        string    `json:"outcome"`
	Digest         string    `json:"digest"`
}

type BlockingItem struct {
	Code     string `json:"code"`
	Message  string `json:"message"`
	Evidence string `json:"evidence,omitempty"`
}
