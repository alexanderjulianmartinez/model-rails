package spec

import (
	"fmt"
	"strings"
)

const (
	SpecVersionV1 = 1
)

const (
	ScopeModel       = "model"
	ScopeVersion     = "version"
	ScopeInteraction = "interaction"
)

const (
	SeverityAllow = "allow"
	SeverityWarn  = "warn"
	SeverityBlock = "block"
)

const (
	ConditionCapabilitySubset = "capability_subset"
	ConditionAllowlist        = "allowlist"
	ConditionDenylist         = "denylist"
	ConditionVersionPin       = "version_pin"
	ConditionEnvironmentGate  = "environment_gate"
	ConditionResourceBounds   = "resource_bounds"
)

type InvariantSpec struct {
	Version    int         `json:"version" yaml:"version"`
	Invariants []Invariant `json:"invariants" yaml:"invariants"`
}

type Invariant struct {
	Name                 string    `json:"name" yaml:"name"`
	Scope                string    `json:"scope" yaml:"scope"`
	Severity             string    `json:"severity" yaml:"severity"`
	Intent               string    `json:"intent" yaml:"intent"`
	Condition            Condition `json:"condition" yaml:"condition"`
	EvidenceRequirements []string  `json:"evidence_requirements,omitempty" yaml:"evidence_requirements,omitempty"`
	Audit                *Audit    `json:"audit,omitempty" yaml:"audit,omitempty"`
}

type Condition struct {
	Type       string                 `json:"type" yaml:"type"`
	Parameters map[string]interface{} `json:"parameters" yaml:"parameters"`
}

type Audit struct {
	Owner     string `json:"owner" yaml:"owner"`
	Rationale string `json:"rationale" yaml:"rationale"`
	Ticket    string `json:"ticket,omitempty" yaml:"ticket,omitempty"`
}

// ValidationError aggregates explicit validation issues.
type ValidationError struct {
	Issues []string
}

func (e *ValidationError) Error() string {
	return "invalid invariant spec: " + strings.Join(e.Issues, "; ")
}

func (s *InvariantSpec) Validate() error {
	var issues []string
	addIssue := func(msg string) {
		issues = append(issues, msg)
	}

	if s.Version != SpecVersionV1 {
		addIssue(fmt.Sprintf("version: expected %d", SpecVersionV1))
	}
	if len(s.Invariants) == 0 {
		addIssue("invariants: must contain at least one invariant")
	}

	nameSet := map[string]struct{}{}
	for idx, inv := range s.Invariants {
		inv.validate(idx, nameSet, addIssue)
	}

	if len(issues) > 0 {
		return &ValidationError{Issues: issues}
	}
	return nil
}

func (i *Invariant) validate(idx int, nameSet map[string]struct{}, addIssue func(string)) {
	prefix := fmt.Sprintf("invariants[%d]", idx)
	if strings.TrimSpace(i.Name) == "" {
		addIssue(prefix + ".name: required")
	} else {
		if _, exists := nameSet[i.Name]; exists {
			addIssue(prefix + ".name: duplicate name")
		} else {
			nameSet[i.Name] = struct{}{}
		}
	}

	if !isValidScope(i.Scope) {
		addIssue(prefix + ".scope: must be model, version, or interaction")
	}
	if !isValidSeverity(i.Severity) {
		addIssue(prefix + ".severity: must be allow, warn, or block")
	}
	if strings.TrimSpace(i.Intent) == "" {
		addIssue(prefix + ".intent: required")
	}

	if strings.TrimSpace(i.Condition.Type) == "" {
		addIssue(prefix + ".condition.type: required")
	} else if !isValidConditionType(i.Condition.Type) {
		addIssue(prefix + ".condition.type: unsupported type")
	}

	if i.Condition.Parameters == nil {
		addIssue(prefix + ".condition.parameters: required")
	} else {
		validateConditionParams(prefix+".condition", i.Condition, addIssue)
	}

	for eIdx, req := range i.EvidenceRequirements {
		if strings.TrimSpace(req) == "" {
			addIssue(fmt.Sprintf("%s.evidence_requirements[%d]: must be non-empty", prefix, eIdx))
		}
	}

	if i.Audit != nil {
		if strings.TrimSpace(i.Audit.Owner) == "" {
			addIssue(prefix + ".audit.owner: required when audit is provided")
		}
		if strings.TrimSpace(i.Audit.Rationale) == "" {
			addIssue(prefix + ".audit.rationale: required when audit is provided")
		}
	}
}

func validateConditionParams(prefix string, c Condition, addIssue func(string)) {
	switch c.Type {
	case ConditionCapabilitySubset:
		validateStringListParam(prefix, c.Parameters, "allowed_capabilities", addIssue)
	case ConditionAllowlist:
		validateStringListParam(prefix, c.Parameters, "allowed_targets", addIssue)
	case ConditionDenylist:
		validateStringListParam(prefix, c.Parameters, "denied_targets", addIssue)
	case ConditionVersionPin:
		allowed, ok, err := getStringSlice(c.Parameters, "allowed_versions")
		exact, okExact, errExact := getString(c.Parameters, "exact_version")
		if err != nil {
			addIssue(prefix + ".parameters.allowed_versions: " + err.Error())
		}
		if errExact != nil {
			addIssue(prefix + ".parameters.exact_version: " + errExact.Error())
		}
		if ok && len(allowed) == 0 {
			addIssue(prefix + ".parameters.allowed_versions: must contain at least one version")
		}
		if okExact && strings.TrimSpace(exact) == "" {
			addIssue(prefix + ".parameters.exact_version: must be non-empty")
		}
		if !ok && !okExact {
			addIssue(prefix + ".parameters: requires allowed_versions or exact_version")
		}
	case ConditionEnvironmentGate:
		validateStringListParam(prefix, c.Parameters, "allowed_environments", addIssue)
	case ConditionResourceBounds:
		requireStringParam(prefix, c.Parameters, "max_cpu", addIssue)
		requireStringParam(prefix, c.Parameters, "max_memory", addIssue)
		requireStringParam(prefix, c.Parameters, "max_gpu", addIssue)
	default:
		addIssue(prefix + ".type: unsupported type")
	}
}

func validateStringListParam(prefix string, params map[string]interface{}, key string, addIssue func(string)) {
	values, ok, err := getStringSlice(params, key)
	if err != nil {
		addIssue(prefix + ".parameters." + key + ": " + err.Error())
		return
	}
	if !ok {
		addIssue(prefix + ".parameters." + key + ": required")
		return
	}
	if len(values) == 0 {
		addIssue(prefix + ".parameters." + key + ": must contain at least one item")
		return
	}
	for idx, v := range values {
		if strings.TrimSpace(v) == "" {
			addIssue(fmt.Sprintf("%s.parameters.%s[%d]: must be non-empty", prefix, key, idx))
		}
	}
}

func requireStringParam(prefix string, params map[string]interface{}, key string, addIssue func(string)) {
	value, ok, err := getString(params, key)
	if err != nil {
		addIssue(prefix + ".parameters." + key + ": " + err.Error())
		return
	}
	if !ok {
		addIssue(prefix + ".parameters." + key + ": required")
		return
	}
	if strings.TrimSpace(value) == "" {
		addIssue(prefix + ".parameters." + key + ": must be non-empty")
		return
	}
}

func getStringSlice(params map[string]interface{}, key string) ([]string, bool, error) {
	val, ok := params[key]
	if !ok {
		return nil, false, nil
	}
	switch v := val.(type) {
	case []string:
		return v, true, nil
	case []interface{}:
		out := make([]string, 0, len(v))
		for idx, item := range v {
			str, ok := item.(string)
			if !ok {
				return nil, true, fmt.Errorf("item %d must be string", idx)
			}
			out = append(out, str)
		}
		return out, true, nil
	default:
		return nil, true, fmt.Errorf("must be a string list")
	}
}

func getString(params map[string]interface{}, key string) (string, bool, error) {
	val, ok := params[key]
	if !ok {
		return "", false, nil
	}
	switch v := val.(type) {
	case string:
		return v, true, nil
	default:
		return "", true, fmt.Errorf("must be a string")
	}
}

func isValidScope(scope string) bool {
	switch scope {
	case ScopeModel, ScopeVersion, ScopeInteraction:
		return true
	default:
		return false
	}
}

func isValidSeverity(sev string) bool {
	switch sev {
	case SeverityAllow, SeverityWarn, SeverityBlock:
		return true
	default:
		return false
	}
}

func isValidConditionType(t string) bool {
	switch t {
	case ConditionCapabilitySubset,
		ConditionAllowlist,
		ConditionDenylist,
		ConditionVersionPin,
		ConditionEnvironmentGate,
		ConditionResourceBounds:
		return true
	default:
		return false
	}
}
