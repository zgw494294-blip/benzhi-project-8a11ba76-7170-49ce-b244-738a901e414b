package domain

import "time"

type auditCanonical struct {
	Sequence       int64  `json:"sequence"`
	CaseID         string `json:"caseId"`
	Action         string `json:"action"`
	Actor          string `json:"actor"`
	OccurredAt     string `json:"occurredAt"`
	PreviousDigest string `json:"previousDigest"`
	PayloadDigest  string `json:"payloadDigest"`
	Outcome        string `json:"outcome"`
}

func NewAuditEvent(sequence int64, caseID, action, actor, previous, payload, outcome string, now time.Time) AuditEvent {
	event := AuditEvent{
		Sequence: sequence, CaseID: caseID, Action: action, Actor: actor,
		OccurredAt: now.UTC(), PreviousDigest: previous, PayloadDigest: payload, Outcome: outcome,
	}
	event.Digest = Digest(auditCanonical{
		Sequence: event.Sequence, CaseID: event.CaseID, Action: event.Action, Actor: event.Actor,
		OccurredAt: event.OccurredAt.Format(time.RFC3339Nano), PreviousDigest: event.PreviousDigest,
		PayloadDigest: event.PayloadDigest, Outcome: event.Outcome,
	})
	return event
}

func ValidateAuditChain(events []AuditEvent) error {
	previous := ""
	for i, event := range events {
		if event.Sequence != int64(i+1) || event.PreviousDigest != previous {
			return NewError("audit_corrupt", "审计序号或前序摘要不连续")
		}
		expected := NewAuditEvent(event.Sequence, event.CaseID, event.Action, event.Actor, event.PreviousDigest, event.PayloadDigest, event.Outcome, event.OccurredAt)
		if expected.Digest != event.Digest {
			return NewError("audit_corrupt", "审计事件摘要不一致")
		}
		previous = event.Digest
	}
	return nil
}
