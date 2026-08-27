package domain

import (
	"sort"
	"strconv"
	"strings"
	"time"
)

type FindingBatchResult struct {
	FindingIDs     []string       `json:"findingIds"`
	Total          int            `json:"total"`
	Resolved       int            `json:"resolved"`
	Open           int            `json:"open"`
	Revision       int64          `json:"revision"`
	CategoryCounts map[string]int `json:"categoryCounts"`
	BlockingItems  []BlockingItem `json:"blockingItems"`
}

func (c *OralHistoryCase) AddFindingsBatch(findings []SensitiveFinding, now time.Time) (FindingBatchResult, error) {
	if err := c.Mutable(); err != nil {
		return FindingBatchResult{}, err
	}
	if len(findings) == 0 {
		return FindingBatchResult{}, Invalid("findings", "至少包含一条敏感项")
	}
	if len(findings) > 100 {
		return FindingBatchResult{}, NewError("findings_batch_too_large", "单批敏感项不得超过 100 条")
	}
	latest := c.LatestTranscript()
	if latest == nil {
		return FindingBatchResult{}, Invalid("transcriptVersionId", "尚未提交转写")
	}
	seenID := map[string]bool{}
	seenTuple := map[string]bool{}
	for i := range findings {
		f := &findings[i]
		if strings.TrimSpace(f.ID) == "" {
			return FindingBatchResult{}, Invalid("findingId", "第"+itoa(i+1)+"条 id 不能为空")
		}
		if seenID[f.ID] {
			return FindingBatchResult{}, NewError("duplicate_finding_id", "第"+itoa(i+1)+"条与批次内其他条目 ID 重复")
		}
		seenID[f.ID] = true
		if strings.TrimSpace(f.Category) == "" || strings.TrimSpace(f.RiskReason) == "" {
			return FindingBatchResult{}, NewError("invalid_finding_item", "第"+itoa(i+1)+"条风险类别和说明不能为空")
		}
		if f.TranscriptVersionID != latest.ID {
			return FindingBatchResult{}, NewError("invalid_transcript_version", "第"+itoa(i+1)+"条必须引用当前最新转写版本")
		}
		found := false
		for _, segment := range latest.Segments {
			if segment.ID == f.SegmentID {
				found = true
				break
			}
		}
		if !found {
			return FindingBatchResult{}, NewError("invalid_segment_id", "第"+itoa(i+1)+"条引用的 segmentId 不存在")
		}
		key := f.SegmentID + "\x00" + strings.TrimSpace(f.Category) + "\x00" + strings.TrimSpace(f.RiskReason)
		if seenTuple[key] {
			return FindingBatchResult{}, NewError("duplicate_finding", "第"+itoa(i+1)+"条与同一片段的类别和风险说明重复")
		}
		seenTuple[key] = true
		for _, existing := range c.Findings {
			if existing.ID == f.ID {
				return FindingBatchResult{}, NewError("conflicting_finding_id", "第"+itoa(i+1)+"条 ID 已存在")
			}
			if existing.TranscriptVersionID == f.TranscriptVersionID && existing.SegmentID == f.SegmentID && existing.Category == strings.TrimSpace(f.Category) && existing.RiskReason == strings.TrimSpace(f.RiskReason) {
				return FindingBatchResult{}, NewError("duplicate_finding", "第"+itoa(i+1)+"条与已有敏感项重复")
			}
		}
	}
	result := FindingBatchResult{Total: len(findings), FindingIDs: make([]string, 0, len(findings)), CategoryCounts: map[string]int{}}
	for _, f := range findings {
		f.CaseID = c.ID
		f.Category = strings.TrimSpace(f.Category)
		f.RiskReason = strings.TrimSpace(f.RiskReason)
		if strings.TrimSpace(f.RedactionProposal) == "" {
			f.Status = FindingOpen
			result.Open++
		} else {
			f.Status = FindingResolved
			f.ResolutionEvidenceRef = latest.ContentDigest
			result.Resolved++
		}
		c.Findings = append(c.Findings, f)
		result.FindingIDs = append(result.FindingIDs, f.ID)
		result.CategoryCounts[f.Category]++
	}
	sort.Strings(result.FindingIDs)
	c.Touch(now)
	c.RecalculateState(now)
	result.Revision = c.Revision
	result.BlockingItems = c.BlockingItems(now)
	return result, nil
}

func itoa(v int) string {
	return strconv.Itoa(v)
}

func (c *OralHistoryCase) AddFinding(finding SensitiveFinding, now time.Time) error {
	if err := c.Mutable(); err != nil {
		return err
	}
	if c.LatestTranscript() == nil {
		return Invalid("transcriptVersionId", "尚未提交转写")
	}
	if strings.TrimSpace(finding.ID) == "" || strings.TrimSpace(finding.Category) == "" || strings.TrimSpace(finding.RiskReason) == "" {
		return Invalid("finding", "id、category 和 riskReason 不能为空")
	}
	version := c.findTranscript(finding.TranscriptVersionID)
	if version == nil || version.ID != c.LatestTranscript().ID {
		return Invalid("transcriptVersionId", "必须引用当前转写版本")
	}
	segmentExists := false
	for _, segment := range version.Segments {
		if segment.ID == finding.SegmentID {
			segmentExists = true
			break
		}
	}
	if !segmentExists {
		return Invalid("segmentId", "未在引用的转写版本中找到")
	}
	for _, existing := range c.Findings {
		if existing.ID == finding.ID {
			return Invalid("findingId", "已经存在")
		}
	}
	finding.CaseID = c.ID
	if strings.TrimSpace(finding.RedactionProposal) == "" {
		finding.Status = FindingOpen
		finding.ResolutionEvidenceRef = ""
	} else {
		finding.Status = FindingResolved
		finding.ResolutionEvidenceRef = version.ContentDigest
	}
	c.Findings = append(c.Findings, finding)
	c.Touch(now)
	c.RecalculateState(now)
	return nil
}

func (c *OralHistoryCase) ReviseFinding(id, proposal, evidenceRef string, now time.Time) error {
	if err := c.Mutable(); err != nil {
		return err
	}
	finding := c.findFinding(id)
	if finding == nil {
		return ErrNotFound
	}
	if strings.TrimSpace(proposal) == "" || !c.ValidRemediationEvidence(evidenceRef) {
		return Invalid("resolutionEvidenceRef", "处置方案必须引用有效的新转写或处置证据")
	}
	finding.RedactionProposal = strings.TrimSpace(proposal)
	finding.ResolutionEvidenceRef = evidenceRef
	finding.Status = FindingResolved
	c.Touch(now)
	c.RecalculateState(now)
	return nil
}

func (c *OralHistoryCase) RaiseObjection(objection Objection, now time.Time) error {
	if err := c.Mutable(); err != nil {
		return err
	}
	if c.LatestTranscript() == nil {
		return Invalid("objection", "没有可复核的转写")
	}
	if strings.TrimSpace(objection.ID) == "" || strings.TrimSpace(objection.Reason) == "" || strings.TrimSpace(objection.RaisedBy) == "" {
		return Invalid("objection", "id、reason 和 raisedBy 不能为空")
	}
	if objection.FindingID != "" && c.findFinding(objection.FindingID) == nil {
		return Invalid("findingId", "引用的敏感项不存在")
	}
	for _, existing := range c.Objections {
		if existing.ID == objection.ID {
			return Invalid("objectionId", "已经存在")
		}
	}
	objection.RaisedAt = now.UTC()
	objection.ClosedAt = nil
	c.Objections = append(c.Objections, objection)
	c.Touch(now)
	c.State = StateRemediation
	return nil
}

func (c *OralHistoryCase) ValidRemediationEvidence(ref string) bool {
	if strings.TrimSpace(ref) == "" {
		return false
	}
	for _, transcript := range c.Transcripts {
		if transcript.ID == ref || transcript.ContentDigest == ref {
			return true
		}
	}
	for _, finding := range c.Findings {
		if finding.ID == ref || finding.ResolutionEvidenceRef == ref {
			return finding.Status == FindingResolved
		}
	}
	return false
}

func (c *OralHistoryCase) CloseObjection(id, evidenceRef, actor string, now time.Time) error {
	if err := c.Mutable(); err != nil {
		return err
	}
	if strings.TrimSpace(actor) == "" {
		return Invalid("actor", "不能为空")
	}
	for i := range c.Objections {
		objection := &c.Objections[i]
		if objection.ID != id {
			continue
		}
		if objection.ClosedAt != nil {
			return Invalid("objection", "已经关闭")
		}
		if !c.ValidRemediationEvidence(evidenceRef) {
			return Invalid("resolutionEvidenceRef", "必须引用有效整改证据")
		}
		t := now.UTC()
		objection.ClosedAt = &t
		objection.ClosedBy = actor
		objection.ResolutionEvidenceRef = evidenceRef
		if objection.FindingID != "" {
			if finding := c.findFinding(objection.FindingID); finding != nil {
				finding.Status = FindingResolved
				finding.ResolutionEvidenceRef = evidenceRef
			}
		}
		c.Touch(now)
		c.RecalculateState(now)
		return nil
	}
	return ErrNotFound
}

func (c *OralHistoryCase) BlockingItems(now time.Time) []BlockingItem {
	var items []BlockingItem
	coverage := c.ConsentCoverage(c.RequestedScope, now)
	if !coverage.Covered {
		items = append(items, BlockingItem{Code: "consent_scope", Message: "有效同意未覆盖拟公开受众和用途"})
		for _, missing := range coverage.Missing {
			items = append(items, BlockingItem{Code: "consent_" + missing.ReasonCode, Message: "同意未覆盖" + missing.Dimension + "「" + missing.Value + "」", Evidence: strings.Join(missing.EvidenceRef, ",")})
		}
	}
	if c.LatestTranscript() == nil {
		items = append(items, BlockingItem{Code: "missing_transcript", Message: "尚未提交转写版本"})
	}
	for _, finding := range c.Findings {
		if finding.Status != FindingResolved {
			items = append(items, BlockingItem{Code: "finding_open", Message: "敏感片段尚未完成处置", Evidence: finding.ID})
		}
	}
	for _, objection := range c.Objections {
		if objection.ClosedAt == nil {
			items = append(items, BlockingItem{Code: "objection_open", Message: "伦理异议尚未闭环", Evidence: objection.ID})
		}
	}
	return items
}

func (c *OralHistoryCase) GovernanceConclusion(now time.Time) string {
	items := c.BlockingItems(now)
	if len(items) == 0 {
		return "全部同意、片段处置与异议证据校验通过，可以冻结。"
	}
	return "存在阻断项，案卷不得冻结或授权。"
}
