package domain

import (
	"sort"
	"strings"
	"time"
)

func validateScope(scope PublicationScope) error {
	if len(normalizeStrings(scope.Audiences)) == 0 {
		return Invalid("audiences", "至少指定一个公开受众")
	}
	if len(normalizeStrings(scope.Purposes)) == 0 {
		return Invalid("purposes", "至少指定一个公开用途")
	}
	return nil
}

func normalizeStrings(items []string) []string {
	set := make(map[string]struct{}, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item != "" {
			set[item] = struct{}{}
		}
	}
	out := make([]string, 0, len(set))
	for item := range set {
		out = append(out, item)
	}
	sort.Strings(out)
	return out
}

func normalizeScope(scope PublicationScope) PublicationScope {
	return PublicationScope{Audiences: normalizeStrings(scope.Audiences), Purposes: normalizeStrings(scope.Purposes)}
}

func containsAll(granted, requested []string) bool {
	set := make(map[string]struct{}, len(granted))
	for _, v := range granted {
		set[v] = struct{}{}
	}
	for _, v := range requested {
		if _, ok := set[v]; !ok {
			return false
		}
	}
	return true
}

func (g ConsentGrant) ActiveAt(now time.Time) bool {
	now = now.UTC()
	return g.WithdrawnAt == nil && !now.Before(g.EffectiveAt) && now.Before(g.ExpiresAt)
}

func (g ConsentGrant) Covers(scope PublicationScope, now time.Time) bool {
	return g.ActiveAt(now) && containsAll(g.AllowedAudiences, scope.Audiences) && containsAll(g.AllowedPurposes, scope.Purposes)
}

func (c *OralHistoryCase) ConsentCovers(scope PublicationScope, now time.Time) bool {
	return c.ConsentCoverage(scope, now).Covered
}

func (c *OralHistoryCase) ConsentCoverage(scope PublicationScope, now time.Time) ConsentCoverage {
	now = now.UTC()
	scope = normalizeScope(scope)
	result := ConsentCoverage{Covered: true, Items: make([]ConsentCoverageItem, 0, len(scope.Audiences)+len(scope.Purposes))}
	for _, group := range []struct {
		dimension string
		values    []string
	}{{"受众", scope.Audiences}, {"用途", scope.Purposes}} {
		for _, value := range group.values {
			item := ConsentCoverageItem{Dimension: group.dimension, Value: value, ReasonCode: "not_registered"}
			var refs []string
			var activeRefs []string
			var earliest *time.Time
			for _, grant := range c.Consents {
				allowed := (group.dimension == "受众" && containsAll(grant.AllowedAudiences, []string{value})) || (group.dimension == "用途" && containsAll(grant.AllowedPurposes, []string{value}))
				if !allowed {
					continue
				}
				refs = append(refs, grant.EvidenceRef)
				if grant.WithdrawnAt != nil {
					if item.ReasonCode == "not_registered" {
						item.ReasonCode = "withdrawn"
					}
					continue
				}
				if now.Before(grant.EffectiveAt) {
					if item.ReasonCode == "not_registered" {
						item.ReasonCode = "not_yet_effective"
					}
					continue
				}
				if !now.Before(grant.ExpiresAt) {
					if item.ReasonCode == "not_registered" || item.ReasonCode == "not_yet_effective" {
						item.ReasonCode = "expired"
					}
					continue
				}
				item.Covered = true
				item.ReasonCode = "covered"
				activeRefs = append(activeRefs, grant.EvidenceRef)
				if earliest == nil || grant.ExpiresAt.Before(*earliest) {
					t := grant.ExpiresAt
					earliest = &t
				}
			}
			sort.Strings(refs)
			if item.Covered {
				sort.Strings(activeRefs)
				item.EvidenceRef = activeRefs
				if earliest != nil && (result.EarliestExpiresAt == nil || earliest.Before(*result.EarliestExpiresAt)) {
					result.EarliestExpiresAt = earliest
				}
			} else {
				item.EvidenceRef = refs
				result.Covered = false
				result.Missing = append(result.Missing, item)
			}
			result.Items = append(result.Items, item)
		}
	}
	if result.EarliestExpiresAt != nil {
		result.RemainingDays = int(result.EarliestExpiresAt.Sub(now) / (24 * time.Hour))
		if result.RemainingDays <= 0 {
			result.Warning = "同意已失效"
		} else if result.RemainingDays <= 30 {
			result.Warning = "有同意将在三十日内到期"
		}
	}
	if result.Warning == "" && !result.Covered {
		for _, missing := range result.Missing {
			if missing.ReasonCode == "expired" || missing.ReasonCode == "withdrawn" {
				result.Warning = "存在已经失效或撤回的同意"
				break
			}
		}
	}
	return result
}

func (c *OralHistoryCase) AddConsent(grant ConsentGrant, now time.Time) error {
	if err := c.Mutable(); err != nil {
		return err
	}
	if strings.TrimSpace(grant.ID) == "" || strings.TrimSpace(grant.EvidenceRef) == "" {
		return Invalid("consent", "id 与 evidenceRef 不能为空")
	}
	if err := validateScope(PublicationScope{Audiences: grant.AllowedAudiences, Purposes: grant.AllowedPurposes}); err != nil {
		return err
	}
	grant.AllowedAudiences = normalizeStrings(grant.AllowedAudiences)
	grant.AllowedPurposes = normalizeStrings(grant.AllowedPurposes)
	grant.CaseID = c.ID
	grant.EffectiveAt = grant.EffectiveAt.UTC()
	grant.ExpiresAt = grant.ExpiresAt.UTC()
	if !grant.ExpiresAt.After(grant.EffectiveAt) {
		return Invalid("expiresAt", "必须晚于 effectiveAt")
	}
	for _, existing := range c.Consents {
		if existing.ID == grant.ID {
			return Invalid("consentId", "已经存在")
		}
	}
	grant.Revision = c.Revision + 1
	c.Consents = append(c.Consents, grant)
	c.Touch(now)
	c.RecalculateState(now)
	return nil
}

func (c *OralHistoryCase) WithdrawConsent(id string, now time.Time) error {
	for i := range c.Consents {
		if c.Consents[i].ID == id {
			if c.Consents[i].WithdrawnAt != nil {
				return Invalid("consent", "已经撤回")
			}
			t := now.UTC()
			c.Consents[i].WithdrawnAt = &t
			for j := range c.Credentials {
				c.Credentials[j].Status = CredentialInvalid
			}
			c.Touch(now)
			c.RecalculateState(now)
			return nil
		}
	}
	return ErrNotFound
}
