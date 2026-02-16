package engine

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/alexanderjulianmartinez/model-rails/internal/spec"
)

const (
	DecisionAllow = "ALLOW"
	DecisionWarn  = "WARN"
	DecisionBlock = "BLOCK"
)

const (
	ActionDeploy  = "deploy"
	ActionUpgrade = "upgrade"
	ActionConnect = "connect"
	ActionPromote = "promote"
)

const (
	EnvDev        = "dev"
	EnvStaging    = "staging"
	EnvProduction = "production"
)

type ActionContext struct {
	ActionType          string `json:"action_type" yaml:"action_type"`
	ModelName           string `json:"model_name" yaml:"model_name"`
	ModelVersion        string `json:"model_version" yaml:"model_version"`
	Environment         string `json:"environment" yaml:"environment"`
	Initiator           string `json:"initiator" yaml:"initiator"`
	Timestamp           string `json:"timestamp" yaml:"timestamp"`
	MetadataSnapshotRef string `json:"metadata_snapshot_ref" yaml:"metadata_snapshot_ref"`
}

func (a ActionContext) Validate() error {
	var issues []string
	add := func(msg string) {
		issues = append(issues, msg)
	}

	if strings.TrimSpace(a.ActionType) == "" {
		add("action.action_type: required")
	} else if !isValidActionType(a.ActionType) {
		add("action.action_type: must be deploy, upgrade, connect, or promote")
	}

	if strings.TrimSpace(a.ModelName) == "" {
		add("action.model_name: required")
	}
	if strings.TrimSpace(a.ModelVersion) == "" {
		add("action.model_version: required")
	}
	if strings.TrimSpace(a.Environment) == "" {
		add("action.environment: required")
	} else if !isValidEnvironment(a.Environment) {
		add("action.environment: must be dev, staging, or production")
	}
	if strings.TrimSpace(a.Initiator) == "" {
		add("action.initiator: required")
	}
	if strings.TrimSpace(a.Timestamp) == "" {
		add("action.timestamp: required")
	} else if _, err := time.Parse(time.RFC3339, a.Timestamp); err != nil {
		add("action.timestamp: must be RFC3339")
	}
	if strings.TrimSpace(a.MetadataSnapshotRef) == "" {
		add("action.metadata_snapshot_ref: required")
	}

	if len(issues) > 0 {
		return &ValidationError{Issues: issues}
	}
	return nil
}

type DependencyModel struct {
	Name    string `json:"name" yaml:"name"`
	Version string `json:"version" yaml:"version"`
}

type ResourceRequirements struct {
	CPU    string `json:"cpu" yaml:"cpu"`
	Memory string `json:"memory" yaml:"memory"`
	GPU    string `json:"gpu" yaml:"gpu"`
}

type ModelMetadata struct {
	ModelName             string               `json:"model_name" yaml:"model_name"`
	ModelVersion          string               `json:"model_version" yaml:"model_version"`
	DeclaredCapabilities  []string             `json:"declared_capabilities" yaml:"declared_capabilities"`
	DependencyModels      []DependencyModel    `json:"dependency_models" yaml:"dependency_models"`
	ResourceRequirements  ResourceRequirements `json:"resource_requirements" yaml:"resource_requirements"`
	TrustDomain           string               `json:"trust_domain" yaml:"trust_domain"`
	DeploymentEnvironment string               `json:"deployment_environment" yaml:"deployment_environment"`
}

type EvidenceItem struct {
	Key   string
	Value interface{}
}

type Violation struct {
	InvariantName string
	Severity      string
	Reason        string
	Evidence      []EvidenceItem
	Remediation   string
}

type DecisionResult struct {
	Decision    string
	Violations  []Violation
	Explanation string
}

type ValidationError struct {
	Issues []string
}

func (e *ValidationError) Error() string {
	return "invalid input: " + strings.Join(e.Issues, "; ")
}

func Evaluate(specDoc spec.InvariantSpec, action ActionContext, metadata ModelMetadata) (DecisionResult, error) {
	if err := specDoc.Validate(); err != nil {
		return DecisionResult{}, err
	}

	if err := validateActionContext(action); err != nil {
		return DecisionResult{}, err
	}

	if err := validateMetadataForSpec(metadata, specDoc.Invariants); err != nil {
		return DecisionResult{}, err
	}

	violations := make([]Violation, 0)
	for _, inv := range specDoc.Invariants {
		ok, violation, err := evaluateInvariant(inv, action, metadata)
		if err != nil {
			return DecisionResult{}, err
		}
		if !ok {
			violations = append(violations, violation)
		}
	}

	sort.SliceStable(violations, func(i, j int) bool {
		if violations[i].InvariantName == violations[j].InvariantName {
			return violations[i].Reason < violations[j].Reason
		}
		return violations[i].InvariantName < violations[j].InvariantName
	})

	decision := aggregateDecision(violations)
	explanation := buildExplanation(decision, violations)

	return DecisionResult{
		Decision:    decision,
		Violations:  violations,
		Explanation: explanation,
	}, nil
}

func validateActionContext(action ActionContext) error {
	return action.Validate()
}

func validateMetadataForSpec(metadata ModelMetadata, invariants []spec.Invariant) error {
	var issues []string
	add := func(msg string) {
		issues = append(issues, msg)
	}

	if strings.TrimSpace(metadata.ModelName) == "" {
		add("metadata.model_name: required")
	}
	if strings.TrimSpace(metadata.ModelVersion) == "" {
		add("metadata.model_version: required")
	}
	if strings.TrimSpace(metadata.TrustDomain) == "" {
		add("metadata.trust_domain: required")
	}
	if strings.TrimSpace(metadata.DeploymentEnvironment) == "" {
		add("metadata.deployment_environment: required")
	}

	requiresCapabilities := false
	requiresDependencies := false
	requiresResource := false

	for _, inv := range invariants {
		switch inv.Condition.Type {
		case spec.ConditionCapabilitySubset:
			requiresCapabilities = true
		case spec.ConditionAllowlist, spec.ConditionDenylist:
			requiresDependencies = true
		case spec.ConditionResourceBounds:
			requiresResource = true
		}
	}

	if requiresCapabilities {
		if len(metadata.DeclaredCapabilities) == 0 {
			add("metadata.declared_capabilities: required")
		}
		for idx, cap := range metadata.DeclaredCapabilities {
			if strings.TrimSpace(cap) == "" {
				add(fmt.Sprintf("metadata.declared_capabilities[%d]: must be non-empty", idx))
			}
		}
	}

	if requiresDependencies {
		if len(metadata.DependencyModels) == 0 {
			add("metadata.dependency_models: required")
		}
		for idx, dep := range metadata.DependencyModels {
			if strings.TrimSpace(dep.Name) == "" {
				add(fmt.Sprintf("metadata.dependency_models[%d].name: required", idx))
			}
			if strings.TrimSpace(dep.Version) == "" {
				add(fmt.Sprintf("metadata.dependency_models[%d].version: required", idx))
			}
		}
	}

	if requiresResource {
		if strings.TrimSpace(metadata.ResourceRequirements.CPU) == "" {
			add("metadata.resource_requirements.cpu: required")
		}
		if strings.TrimSpace(metadata.ResourceRequirements.Memory) == "" {
			add("metadata.resource_requirements.memory: required")
		}
		if strings.TrimSpace(metadata.ResourceRequirements.GPU) == "" {
			add("metadata.resource_requirements.gpu: required")
		}
	}

	if len(issues) > 0 {
		return &ValidationError{Issues: issues}
	}
	return nil
}

func evaluateInvariant(inv spec.Invariant, action ActionContext, metadata ModelMetadata) (bool, Violation, error) {
	violation := Violation{
		InvariantName: inv.Name,
		Severity:      strings.ToUpper(inv.Severity),
		Remediation:   remediationFor(inv.Condition.Type),
	}

	switch inv.Condition.Type {
	case spec.ConditionCapabilitySubset:
		allowed, err := requiredStringSlice(inv.Condition.Parameters, "allowed_capabilities")
		if err != nil {
			return false, violation, err
		}
		missing := diff(metadata.DeclaredCapabilities, allowed)
		if len(missing) == 0 {
			return true, Violation{}, nil
		}
		violation.Reason = "Declared capabilities exceed allowed set."
		violation.Evidence = sortEvidence([]EvidenceItem{
			{Key: "allowed_capabilities", Value: allowed},
			{Key: "observed_capabilities", Value: metadata.DeclaredCapabilities},
		})
		return false, violation, nil
	case spec.ConditionAllowlist:
		allowed, err := requiredStringSlice(inv.Condition.Parameters, "allowed_targets")
		if err != nil {
			return false, violation, err
		}
		observed := dependencyNames(metadata.DependencyModels)
		blocked := diff(observed, allowed)
		if len(blocked) == 0 {
			return true, Violation{}, nil
		}
		violation.Reason = "Observed dependency targets are not in allowlist."
		violation.Evidence = sortEvidence([]EvidenceItem{
			{Key: "allowed_targets", Value: allowed},
			{Key: "observed_targets", Value: observed},
		})
		return false, violation, nil
	case spec.ConditionDenylist:
		denied, err := requiredStringSlice(inv.Condition.Parameters, "denied_targets")
		if err != nil {
			return false, violation, err
		}
		observed := dependencyNames(metadata.DependencyModels)
		found := intersect(observed, denied)
		if len(found) == 0 {
			return true, Violation{}, nil
		}
		violation.Reason = "Observed dependency targets are denied."
		violation.Evidence = sortEvidence([]EvidenceItem{
			{Key: "denied_targets", Value: denied},
			{Key: "observed_targets", Value: observed},
		})
		return false, violation, nil
	case spec.ConditionVersionPin:
		allowed, okAllowed, err := optionalStringSlice(inv.Condition.Parameters, "allowed_versions")
		if err != nil {
			return false, violation, err
		}
		exact, okExact, err := optionalString(inv.Condition.Parameters, "exact_version")
		if err != nil {
			return false, violation, err
		}
		if okExact {
			if metadata.ModelVersion == exact {
				return true, Violation{}, nil
			}
			violation.Reason = "Model version does not match exact pin."
			violation.Evidence = sortEvidence([]EvidenceItem{
				{Key: "exact_version", Value: exact},
				{Key: "observed_version", Value: metadata.ModelVersion},
			})
			return false, violation, nil
		}
		if okAllowed {
			if contains(allowed, metadata.ModelVersion) {
				return true, Violation{}, nil
			}
			violation.Reason = "Model version is not in allowed set."
			violation.Evidence = sortEvidence([]EvidenceItem{
				{Key: "allowed_versions", Value: allowed},
				{Key: "observed_version", Value: metadata.ModelVersion},
			})
			return false, violation, nil
		}
		return false, violation, fmt.Errorf("invariant %s: condition parameters require allowed_versions or exact_version", inv.Name)
	case spec.ConditionEnvironmentGate:
		allowed, err := requiredStringSlice(inv.Condition.Parameters, "allowed_environments")
		if err != nil {
			return false, violation, err
		}
		if contains(allowed, action.Environment) {
			return true, Violation{}, nil
		}
		violation.Reason = "Action environment is not in allowed set."
		violation.Evidence = sortEvidence([]EvidenceItem{
			{Key: "allowed_environments", Value: allowed},
			{Key: "observed_environment", Value: action.Environment},
		})
		return false, violation, nil
	case spec.ConditionResourceBounds:
		maxCPU, err := requiredString(inv.Condition.Parameters, "max_cpu")
		if err != nil {
			return false, violation, err
		}
		maxMemory, err := requiredString(inv.Condition.Parameters, "max_memory")
		if err != nil {
			return false, violation, err
		}
		maxGPU, err := requiredString(inv.Condition.Parameters, "max_gpu")
		if err != nil {
			return false, violation, err
		}

		obsCPU, err := parseCPU(metadata.ResourceRequirements.CPU)
		if err != nil {
			return false, violation, err
		}
		obsMem, err := parseMemoryGi(metadata.ResourceRequirements.Memory)
		if err != nil {
			return false, violation, err
		}
		obsGPU, err := parseGPU(metadata.ResourceRequirements.GPU)
		if err != nil {
			return false, violation, err
		}

		limitCPU, err := parseCPU(maxCPU)
		if err != nil {
			return false, violation, err
		}
		limitMem, err := parseMemoryGi(maxMemory)
		if err != nil {
			return false, violation, err
		}
		limitGPU, err := parseGPU(maxGPU)
		if err != nil {
			return false, violation, err
		}

		if obsCPU <= limitCPU && obsMem <= limitMem && obsGPU <= limitGPU {
			return true, Violation{}, nil
		}

		violation.Reason = "Resource requirements exceed declared bounds."
		violation.Evidence = sortEvidence([]EvidenceItem{
			{Key: "max_cpu", Value: maxCPU},
			{Key: "max_memory", Value: maxMemory},
			{Key: "max_gpu", Value: maxGPU},
			{Key: "observed_cpu", Value: metadata.ResourceRequirements.CPU},
			{Key: "observed_memory", Value: metadata.ResourceRequirements.Memory},
			{Key: "observed_gpu", Value: metadata.ResourceRequirements.GPU},
		})
		return false, violation, nil
	default:
		return false, violation, fmt.Errorf("invariant %s: unsupported condition type %s", inv.Name, inv.Condition.Type)
	}
}

func aggregateDecision(violations []Violation) string {
	decision := DecisionAllow
	for _, v := range violations {
		switch strings.ToLower(v.Severity) {
		case spec.SeverityBlock:
			return DecisionBlock
		case spec.SeverityWarn:
			decision = DecisionWarn
		}
	}
	return decision
}

func buildExplanation(decision string, violations []Violation) string {
	switch decision {
	case DecisionAllow:
		return "All invariants satisfied."
	case DecisionWarn:
		return formatDecisionDetails("WARN", violations)
	case DecisionBlock:
		return formatDecisionDetails("BLOCK", violations)
	default:
		return "Decision evaluation completed."
	}
}

func formatDecisionDetails(severity string, violations []Violation) string {
	if len(violations) == 0 {
		return "No invariant violations."
	}
	lines := make([]string, 0, len(violations)+1)
	lines = append(lines, fmt.Sprintf("%d invariant(s) violated with %s severity.", len(violations), severity))
	for idx, v := range violations {
		lines = append(lines, fmt.Sprintf(
			"violation[%d]: invariant=%s severity=%s reason=%s remediation=%s evidence=%s",
			idx+1,
			v.InvariantName,
			v.Severity,
			v.Reason,
			v.Remediation,
			formatEvidence(v.Evidence),
		))
	}
	return strings.Join(lines, "\n")
}

func remediationFor(conditionType string) string {
	switch conditionType {
	case spec.ConditionCapabilitySubset:
		return "Remove disallowed capabilities or update allowed_capabilities in the spec."
	case spec.ConditionAllowlist:
		return "Remove unapproved targets or add them to allowed_targets in the spec."
	case spec.ConditionDenylist:
		return "Remove denied targets or update denied_targets in the spec."
	case spec.ConditionVersionPin:
		return "Use an allowed version or update allowed_versions/exact_version in the spec."
	case spec.ConditionEnvironmentGate:
		return "Use an allowed environment or update allowed_environments in the spec."
	case spec.ConditionResourceBounds:
		return "Reduce resource requirements or update max_* bounds in the spec."
	default:
		return "Update the invariant definition or input data to satisfy this condition."
	}
}

func formatEvidence(items []EvidenceItem) string {
	if len(items) == 0 {
		return "none"
	}
	parts := make([]string, 0, len(items))
	for _, item := range items {
		parts = append(parts, fmt.Sprintf("%s=%v", item.Key, item.Value))
	}
	return strings.Join(parts, ", ")
}

func requiredStringSlice(params map[string]interface{}, key string) ([]string, error) {
	values, ok, err := optionalStringSlice(params, key)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("condition.parameters.%s: required", key)
	}
	if len(values) == 0 {
		return nil, fmt.Errorf("condition.parameters.%s: must contain at least one item", key)
	}
	for idx, v := range values {
		if strings.TrimSpace(v) == "" {
			return nil, fmt.Errorf("condition.parameters.%s[%d]: must be non-empty", key, idx)
		}
	}
	return values, nil
}

func requiredString(params map[string]interface{}, key string) (string, error) {
	value, ok, err := optionalString(params, key)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", fmt.Errorf("condition.parameters.%s: required", key)
	}
	if strings.TrimSpace(value) == "" {
		return "", fmt.Errorf("condition.parameters.%s: must be non-empty", key)
	}
	return value, nil
}

func optionalStringSlice(params map[string]interface{}, key string) ([]string, bool, error) {
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
				return nil, true, fmt.Errorf("condition.parameters.%s[%d]: must be string", key, idx)
			}
			out = append(out, str)
		}
		return out, true, nil
	default:
		return nil, true, fmt.Errorf("condition.parameters.%s: must be a string list", key)
	}
}

func optionalString(params map[string]interface{}, key string) (string, bool, error) {
	val, ok := params[key]
	if !ok {
		return "", false, nil
	}
	switch v := val.(type) {
	case string:
		return v, true, nil
	default:
		return "", true, fmt.Errorf("condition.parameters.%s: must be a string", key)
	}
}

func parseCPU(raw string) (float64, error) {
	value, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	if err != nil {
		return 0, fmt.Errorf("cpu value must be numeric")
	}
	return value, nil
}

func parseGPU(raw string) (int, error) {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return 0, fmt.Errorf("gpu value must be integer")
	}
	return value, nil
}

func parseMemoryGi(raw string) (int, error) {
	value := strings.TrimSpace(strings.ToLower(raw))
	value = strings.TrimSuffix(value, "gi")
	value = strings.TrimSuffix(value, "g")
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("memory value must be an integer with Gi suffix")
	}
	return parsed, nil
}

func diff(observed []string, allowed []string) []string {
	allowSet := map[string]struct{}{}
	for _, v := range allowed {
		allowSet[v] = struct{}{}
	}
	var out []string
	for _, v := range observed {
		if _, ok := allowSet[v]; !ok {
			out = append(out, v)
		}
	}
	return uniqueSorted(out)
}

func intersect(a []string, b []string) []string {
	bSet := map[string]struct{}{}
	for _, v := range b {
		bSet[v] = struct{}{}
	}
	var out []string
	for _, v := range a {
		if _, ok := bSet[v]; ok {
			out = append(out, v)
		}
	}
	return uniqueSorted(out)
}

func contains(list []string, target string) bool {
	for _, v := range list {
		if v == target {
			return true
		}
	}
	return false
}

func dependencyNames(deps []DependencyModel) []string {
	out := make([]string, 0, len(deps))
	for _, d := range deps {
		if strings.TrimSpace(d.Name) != "" {
			out = append(out, d.Name)
		}
	}
	return uniqueSorted(out)
}

func uniqueSorted(values []string) []string {
	set := map[string]struct{}{}
	for _, v := range values {
		set[v] = struct{}{}
	}
	out := make([]string, 0, len(set))
	for v := range set {
		out = append(out, v)
	}
	sort.Strings(out)
	return out
}

func sortEvidence(items []EvidenceItem) []EvidenceItem {
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].Key < items[j].Key
	})
	return items
}

func isValidActionType(actionType string) bool {
	switch actionType {
	case ActionDeploy, ActionUpgrade, ActionConnect, ActionPromote:
		return true
	default:
		return false
	}
}

func isValidEnvironment(env string) bool {
	switch env {
	case EnvDev, EnvStaging, EnvProduction:
		return true
	default:
		return false
	}
}
