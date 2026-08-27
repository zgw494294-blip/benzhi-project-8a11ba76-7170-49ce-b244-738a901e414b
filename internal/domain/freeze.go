package domain

import (
	"sort"
	"strings"
	"time"
)

type manifestCanonical struct {
	CaseID              string   `json:"caseId"`
	TranscriptVersionID string   `json:"transcriptVersionId"`
	EvidenceDigests     []string `json:"evidenceDigests"`
	ContentDigest       string   `json:"contentDigest"`
	RuleVersion         string   `json:"ruleVersion"`
}

func (c *OralHistoryCase) FreezePreview(now time.Time) FreezePreview {
	preview := FreezePreview{RuleVersion: RuleVersion, Evidence: []ManifestEvidence{}, BlockingItems: c.BlockingItems(now)}
	latest := c.LatestTranscript()
	if latest == nil {
		return preview
	}
	preview.TranscriptVersionID, preview.ContentDigest = latest.ID, latest.ContentDigest
	preview.Evidence = append(preview.Evidence, ManifestEvidence{Kind: "transcript", ID: latest.ID, Digest: latest.ContentDigest, Summary: "最新转写版本"})
	for _, consent := range c.Consents {
		preview.Evidence = append(preview.Evidence, ManifestEvidence{Kind: "consent", ID: consent.ID, Digest: Digest(consent), Summary: consent.EvidenceRef})
	}
	for _, finding := range c.Findings {
		preview.Evidence = append(preview.Evidence, ManifestEvidence{Kind: "finding", ID: finding.ID, Digest: Digest(finding), Summary: finding.Category + "：" + finding.RiskReason})
	}
	for _, objection := range c.Objections {
		preview.Evidence = append(preview.Evidence, ManifestEvidence{Kind: "objection", ID: objection.ID, Digest: Digest(objection), Summary: objection.Reason})
	}
	sort.Slice(preview.Evidence, func(i, j int) bool {
		if preview.Evidence[i].Kind == preview.Evidence[j].Kind {
			return preview.Evidence[i].ID < preview.Evidence[j].ID
		}
		return preview.Evidence[i].Kind < preview.Evidence[j].Kind
	})
	evidence := make([]string, 0, len(preview.Evidence))
	for _, item := range preview.Evidence {
		evidence = append(evidence, item.Digest)
	}
	sort.Strings(evidence)
	preview.ManifestDigest = Digest(manifestCanonical{CaseID: c.ID, TranscriptVersionID: latest.ID, EvidenceDigests: evidence, ContentDigest: latest.ContentDigest, RuleVersion: RuleVersion})
	if len(preview.BlockingItems) > 0 {
		preview.ManifestDigest = ""
	}
	return preview
}

func (c *OralHistoryCase) Freeze(id, actor string, now time.Time) (*FrozenManifest, error) {
	return c.freeze(id, actor, now, "")
}

func (c *OralHistoryCase) FreezeConfirmed(id, actor, confirmedDigest string, now time.Time) (*FrozenManifest, error) {
	return c.freeze(id, actor, now, confirmedDigest)
}

func (c *OralHistoryCase) freeze(id, actor string, now time.Time, confirmedDigest string) (*FrozenManifest, error) {
	if err := c.Mutable(); err != nil {
		return nil, err
	}
	if strings.TrimSpace(id) == "" || strings.TrimSpace(actor) == "" {
		return nil, Invalid("freeze", "id 与 actor 不能为空")
	}
	blocks := c.BlockingItems(now)
	if len(blocks) > 0 {
		return nil, NewError("freeze_blocked", blocks[0].Message)
	}
	preview := c.FreezePreview(now)
	if confirmedDigest != "" && confirmedDigest != preview.ManifestDigest {
		return nil, NewError("manifest_conflict", "候选摘要已变化，请重新预览并确认")
	}
	latest := c.LatestTranscript()
	evidence := make([]string, 0, len(preview.Evidence))
	for _, item := range preview.Evidence {
		evidence = append(evidence, item.Digest)
	}
	sort.Strings(evidence)
	canonical := manifestCanonical{
		CaseID: c.ID, TranscriptVersionID: latest.ID, EvidenceDigests: evidence,
		ContentDigest: latest.ContentDigest, RuleVersion: RuleVersion,
	}
	manifest := &FrozenManifest{
		ID: id, CaseID: c.ID, TranscriptVersionID: latest.ID, EvidenceDigests: evidence,
		ContentDigest: latest.ContentDigest, RuleVersion: RuleVersion,
		ManifestDigest: Digest(canonical), FrozenBy: actor, FrozenAt: now.UTC(),
	}
	c.FrozenManifest = manifest
	c.State = StateFrozen
	c.Touch(now)
	return manifest, nil
}

func RecalculateManifestDigest(manifest FrozenManifest) string {
	return Digest(manifestCanonical{
		CaseID: manifest.CaseID, TranscriptVersionID: manifest.TranscriptVersionID,
		EvidenceDigests: manifest.EvidenceDigests, ContentDigest: manifest.ContentDigest,
		RuleVersion: manifest.RuleVersion,
	})
}
