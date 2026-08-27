package domain

import (
	"testing"
	"time"
)

func TestCompleteGovernanceWorkflow(t *testing.T) {
	now := time.Date(2026, 8, 27, 8, 0, 0, 0, time.UTC)
	caseFile, err := NewCase(CreateCaseInput{
		ID: "case-1", Title: "技艺口述", CollectionUnit: "市档案馆", IntervieweeCode: "A-01", SourceRef: "audio://1",
		RequestedScope: PublicationScope{Audiences: []string{"公众"}, Purposes: []string{"研究"}}, Now: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	if caseFile.State != StateDraft || caseFile.Revision != 1 {
		t.Fatalf("unexpected initial state: %+v", caseFile)
	}
	err = caseFile.AddConsent(ConsentGrant{
		ID: "c1", EvidenceRef: "signed://1", AllowedAudiences: []string{"公众"}, AllowedPurposes: []string{"研究"},
		EffectiveAt: now.Add(-time.Hour), ExpiresAt: now.Add(time.Hour),
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	if caseFile.State != StateConsentVerified {
		t.Fatalf("want consent verified, got %s", caseFile.State)
	}
	if err := caseFile.SubmitTranscript(TranscriptVersion{ID: "v1", SubmittedBy: "整理员", Segments: []TranscriptSegment{{ID: "s1", StartMS: 0, EndMS: 1000, Text: "姓名张某"}}}, now); err != nil {
		t.Fatal(err)
	}
	if err := caseFile.AddFinding(SensitiveFinding{ID: "f1", TranscriptVersionID: "v1", SegmentID: "s1", Category: "隐私", RiskReason: "姓名", RedactionProposal: "使用代号"}, now); err != nil {
		t.Fatal(err)
	}
	if caseFile.State != StateFreezable {
		t.Fatalf("want freezable, got %s", caseFile.State)
	}
	if err := caseFile.RaiseObjection(Objection{ID: "o1", Reason: "需固化文本", FindingID: "f1", RaisedBy: "复核员"}, now); err != nil {
		t.Fatal(err)
	}
	if caseFile.State != StateRemediation {
		t.Fatalf("want remediation, got %s", caseFile.State)
	}
	if err := caseFile.CloseObjection("o1", "missing", "复核员", now); err == nil {
		t.Fatal("invalid evidence should be rejected")
	}
	if caseFile.Objections[0].ClosedAt != nil {
		t.Fatal("failed close mutated objection")
	}
	if err := caseFile.SubmitTranscript(TranscriptVersion{ID: "v2", BaseVersionID: "v1", SubmittedBy: "整理员", Segments: []TranscriptSegment{{ID: "s1", StartMS: 0, EndMS: 1000, Text: "受访者本人"}}}, now); err != nil {
		t.Fatal(err)
	}
	if err := caseFile.CloseObjection("o1", "v2", "复核员", now); err != nil {
		t.Fatal(err)
	}
	manifest, err := caseFile.Freeze("m1", "整理员", now)
	if err != nil {
		t.Fatal(err)
	}
	if RecalculateManifestDigest(*manifest) != manifest.ManifestDigest {
		t.Fatal("manifest digest is unstable")
	}
	credential, err := caseFile.Authorize("r1", caseFile.RequestedScope, "负责人", now)
	if err != nil {
		t.Fatal(err)
	}
	if verification := caseFile.VerifyCredential(credential.ID, now); !verification.Valid {
		t.Fatalf("credential invalid: %+v", verification)
	}
	if err := caseFile.WithdrawConsent("c1", now); err != nil {
		t.Fatal(err)
	}
	if verification := caseFile.VerifyCredential(credential.ID, now); verification.Valid {
		t.Fatal("withdrawal should invalidate credential")
	}
}

func TestTranscriptValidation(t *testing.T) {
	tests := []struct {
		name     string
		segments []TranscriptSegment
	}{
		{"empty", nil},
		{"overlap", []TranscriptSegment{{ID: "a", StartMS: 0, EndMS: 20, Text: "a"}, {ID: "b", StartMS: 10, EndMS: 30, Text: "b"}}},
		{"duplicate", []TranscriptSegment{{ID: "a", StartMS: 0, EndMS: 10, Text: "a"}, {ID: "a", StartMS: 10, EndMS: 20, Text: "b"}}},
		{"blank", []TranscriptSegment{{ID: "a", StartMS: 0, EndMS: 10, Text: " "}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if validateSegments(test.segments) == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestPartialConsentIsRecordedButBlocksTranscript(t *testing.T) {
	now := time.Now().UTC()
	caseFile, err := NewCase(CreateCaseInput{
		ID: "partial", Title: "资料", CollectionUnit: "馆", IntervieweeCode: "P", SourceRef: "source",
		RequestedScope: PublicationScope{Audiences: []string{"公众", "研究者"}, Purposes: []string{"研究"}}, Now: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	err = caseFile.AddConsent(ConsentGrant{
		ID: "public", EvidenceRef: "proof", AllowedAudiences: []string{"公众"}, AllowedPurposes: []string{"研究"},
		EffectiveAt: now.Add(-time.Hour), ExpiresAt: now.Add(time.Hour),
	}, now)
	if err != nil || len(caseFile.Consents) != 1 || caseFile.State != StateDraft {
		t.Fatalf("partial grant should remain recorded in draft: err=%v case=%+v", err, caseFile)
	}
	err = caseFile.SubmitTranscript(TranscriptVersion{ID: "v1", SubmittedBy: "整理员", Segments: []TranscriptSegment{{ID: "s", StartMS: 0, EndMS: 1, Text: "文本"}}}, now)
	if ErrorCode(err) != "consent_blocked" {
		t.Fatalf("want consent block, got %v", err)
	}
}

func TestAuditChainDetectsTampering(t *testing.T) {
	now := time.Now().UTC()
	first := NewAuditEvent(1, "case", "create", "actor", "", "payload", "success", now)
	second := NewAuditEvent(2, "case", "update", "actor", first.Digest, "next", "success", now)
	if err := ValidateAuditChain([]AuditEvent{first, second}); err != nil {
		t.Fatal(err)
	}
	second.PayloadDigest = "changed"
	if err := ValidateAuditChain([]AuditEvent{first, second}); err == nil {
		t.Fatal("tampered chain must fail")
	}
}
