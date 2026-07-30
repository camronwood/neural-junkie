package intent

import (
	"sort"
	"strings"
	"sync"
)

// CoreDomains are always valid for semantic domain stamps (nil registry fallback).
var CoreDomains = []string{
	"general", "security", "biology", "frontend", "backend", "devops",
	"architecture", "code_review", "database", "rust", "cad",
}

// CoreRecipients are always valid for recipient_type stamps (nil registry fallback).
var CoreRecipients = []string{
	"general", "assistant", "frontend", "backend", "devops", "architecture",
	"code-review", "database", "security", "biology", "rust", "cad",
}

// OntologyRegistry is the typed domain/recipient vocabulary for semantic validation
// and classifier enums. Packs extend it via enabled agent types — this is not a
// model/cost router.
type OntologyRegistry struct {
	Domains    map[string]struct{}
	Recipients map[string]struct{}
}

var (
	ontologyMu  sync.RWMutex
	ontologyReg *OntologyRegistry
)

// SetOntologyRegistry installs the process-wide ontology (nil restores core defaults).
func SetOntologyRegistry(reg *OntologyRegistry) {
	ontologyMu.Lock()
	defer ontologyMu.Unlock()
	ontologyReg = reg
}

// CurrentOntology returns the active registry, or a core-only copy when unset.
func CurrentOntology() *OntologyRegistry {
	ontologyMu.RLock()
	reg := ontologyReg
	ontologyMu.RUnlock()
	if reg != nil {
		return reg
	}
	return NewOntologyRegistry(nil, nil)
}

// NewOntologyRegistry builds a registry from optional pack agent types plus core defaults.
// Agent types become both domain (underscored) and recipient (hyphenated when needed).
func NewOntologyRegistry(domains, recipients []string) *OntologyRegistry {
	reg := &OntologyRegistry{
		Domains:    make(map[string]struct{}, len(CoreDomains)+len(domains)),
		Recipients: make(map[string]struct{}, len(CoreRecipients)+len(recipients)),
	}
	for _, d := range CoreDomains {
		reg.Domains[d] = struct{}{}
	}
	for _, r := range CoreRecipients {
		reg.Recipients[r] = struct{}{}
	}
	for _, d := range domains {
		d = normalizeDomainToken(d)
		if d != "" {
			reg.Domains[d] = struct{}{}
		}
	}
	for _, r := range recipients {
		r = normalizeRecipientToken(r)
		if r != "" {
			reg.Recipients[r] = struct{}{}
		}
	}
	return reg
}

// OntologyFromAgentTypes derives domain + recipient tokens from pack/agent type strings.
func OntologyFromAgentTypes(agentTypes []string) *OntologyRegistry {
	domains := make([]string, 0, len(agentTypes))
	recipients := make([]string, 0, len(agentTypes))
	for _, t := range agentTypes {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		domains = append(domains, normalizeDomainToken(t))
		recipients = append(recipients, normalizeRecipientToken(t))
		// code_review domain pairs with code-review recipient historically.
		if normalizeDomainToken(t) == "code_review" {
			recipients = append(recipients, "code-review")
		}
		if normalizeRecipientToken(t) == "code-review" {
			domains = append(domains, "code_review")
		}
	}
	return NewOntologyRegistry(domains, recipients)
}

func (r *OntologyRegistry) ValidDomain(value string) bool {
	if strings.TrimSpace(value) == "" {
		return true
	}
	if r == nil {
		return slicesContains(CoreDomains, normalizeDomainToken(value))
	}
	_, ok := r.Domains[normalizeDomainToken(value)]
	return ok
}

func (r *OntologyRegistry) ValidRecipient(value string) bool {
	if strings.TrimSpace(value) == "" {
		return true
	}
	if r == nil {
		return slicesContains(CoreRecipients, normalizeRecipientToken(value))
	}
	_, ok := r.Recipients[normalizeRecipientToken(value)]
	return ok
}

// DomainEnum returns sorted domain tokens for prompt/schema interpolation.
func (r *OntologyRegistry) DomainEnum() []string {
	if r == nil {
		out := append([]string(nil), CoreDomains...)
		sort.Strings(out)
		return out
	}
	out := make([]string, 0, len(r.Domains))
	for d := range r.Domains {
		out = append(out, d)
	}
	sort.Strings(out)
	return out
}

// RecipientEnum returns sorted recipient tokens for prompt/schema interpolation.
func (r *OntologyRegistry) RecipientEnum() []string {
	if r == nil {
		out := append([]string(nil), CoreRecipients...)
		sort.Strings(out)
		return out
	}
	out := make([]string, 0, len(r.Recipients))
	for rec := range r.Recipients {
		out = append(out, rec)
	}
	sort.Strings(out)
	return out
}

func normalizeDomainToken(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "-", "_")
	value = strings.ReplaceAll(value, " ", "_")
	return value
}

func normalizeRecipientToken(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "_", "-")
	value = strings.ReplaceAll(value, " ", "-")
	// Historical: domain uses code_review, recipient uses code-review.
	if value == "codereview" {
		return "code-review"
	}
	return value
}

func slicesContains(list []string, value string) bool {
	for _, item := range list {
		if item == value {
			return true
		}
	}
	return false
}
