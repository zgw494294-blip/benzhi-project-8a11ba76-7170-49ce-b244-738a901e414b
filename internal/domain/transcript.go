package domain

import (
	"sort"
	"strings"
	"time"
)

func validateSegments(segments []TranscriptSegment) error {
	if len(segments) == 0 {
		return Invalid("segments", "至少包含一个片段")
	}
	seen := make(map[string]bool, len(segments))
	var previousEnd int64 = -1
	for i, segment := range segments {
		if strings.TrimSpace(segment.ID) == "" || seen[segment.ID] {
			return Invalid("segmentId", "不能为空且不得重复")
		}
		seen[segment.ID] = true
		if segment.StartMS < 0 || segment.EndMS <= segment.StartMS {
			return Invalid("segmentTime", "结束时间必须大于非负开始时间")
		}
		if i > 0 && segment.StartMS < previousEnd {
			return Invalid("segmentOrder", "片段必须按时间排列且不得重叠")
		}
		if strings.TrimSpace(segment.Text) == "" {
			return Invalid("segmentText", "不能为空")
		}
		previousEnd = segment.EndMS
	}
	return nil
}

func (c *OralHistoryCase) SubmitTranscript(version TranscriptVersion, now time.Time) error {
	if err := c.Mutable(); err != nil {
		return err
	}
	if !c.ConsentCovers(c.RequestedScope, now) {
		return ErrConsentBlocked
	}
	if strings.TrimSpace(version.ID) == "" || strings.TrimSpace(version.SubmittedBy) == "" {
		return Invalid("transcript", "id 与 submittedBy 不能为空")
	}
	if err := validateSegments(version.Segments); err != nil {
		return err
	}
	latest := c.LatestTranscript()
	if latest == nil && version.BaseVersionID != "" {
		return Invalid("baseVersionId", "首个版本不得指定基线")
	}
	if latest != nil && version.BaseVersionID != latest.ID {
		return Invalid("baseVersionId", "必须引用当前最新转写版本")
	}
	for _, existing := range c.Transcripts {
		if existing.ID == version.ID {
			return Invalid("transcriptId", "已经存在")
		}
	}
	version.CaseID = c.ID
	version.SubmittedAt = now.UTC()
	version.ContentDigest = Digest(version.Segments)
	if latest != nil {
		for _, diff := range DiffTranscripts(latest, &version) {
			if diff.Kind == "新增" {
				continue
			}
			for i := range c.Findings {
				finding := &c.Findings[i]
				if finding.TranscriptVersionID == latest.ID && finding.SegmentID == diff.SegmentID {
					finding.PriorResolutionEvidenceRef = finding.ResolutionEvidenceRef
					finding.Status = FindingOpen
					finding.ResolutionEvidenceRef = ""
				}
			}
		}
	}
	c.Transcripts = append(c.Transcripts, version)
	c.Touch(now)
	c.RecalculateState(now)
	return nil
}

type SegmentDifference struct {
	SegmentID     string   `json:"segmentId"`
	Kind          string   `json:"kind"`
	Before        string   `json:"before,omitempty"`
	After         string   `json:"after,omitempty"`
	BeforeStartMS int64    `json:"beforeStartMs"`
	BeforeEndMS   int64    `json:"beforeEndMs"`
	AfterStartMS  int64    `json:"afterStartMs"`
	AfterEndMS    int64    `json:"afterEndMs"`
	FindingIDs    []string `json:"findingIds,omitempty"`
}

func DiffTranscripts(base, current *TranscriptVersion) []SegmentDifference {
	if current == nil {
		return nil
	}
	type segmentSnapshot struct {
		text       string
		start, end int64
	}
	before := map[string]segmentSnapshot{}
	if base != nil {
		for _, s := range base.Segments {
			before[s.ID] = segmentSnapshot{text: s.Text, start: s.StartMS, end: s.EndMS}
		}
	}
	var differences []SegmentDifference
	for _, s := range current.Segments {
		old, exists := before[s.ID]
		switch {
		case !exists:
			differences = append(differences, SegmentDifference{SegmentID: s.ID, Kind: "新增", After: s.Text, AfterStartMS: s.StartMS, AfterEndMS: s.EndMS})
		default:
			textChanged := old.text != s.Text
			timeChanged := old.start != s.StartMS || old.end != s.EndMS
			if textChanged || timeChanged {
				kind := "改时"
				if textChanged && timeChanged {
					kind = "改文改时"
				} else if textChanged {
					kind = "改文"
				}
				differences = append(differences, SegmentDifference{SegmentID: s.ID, Kind: kind, Before: old.text, After: s.Text, BeforeStartMS: old.start, BeforeEndMS: old.end, AfterStartMS: s.StartMS, AfterEndMS: s.EndMS})
			}
		}
		delete(before, s.ID)
	}
	for id, old := range before {
		differences = append(differences, SegmentDifference{SegmentID: id, Kind: "删除", Before: old.text, BeforeStartMS: old.start, BeforeEndMS: old.end})
	}
	sort.SliceStable(differences, func(i, j int) bool {
		start := func(d SegmentDifference) int64 {
			if d.AfterStartMS != 0 || d.Kind == "新增" {
				return d.AfterStartMS
			}
			return d.BeforeStartMS
		}
		if start(differences[i]) == start(differences[j]) {
			return differences[i].SegmentID < differences[j].SegmentID
		}
		return start(differences[i]) < start(differences[j])
	})
	return differences
}
