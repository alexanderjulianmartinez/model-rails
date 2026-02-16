package engine

import "time"

// InteractionMetadata represents a model-to-model interaction at evaluation time.
// It is immutable input used for interaction-scoped invariants.
type InteractionMetadata struct {
	SourceModelName     string `json:"source_model_name" yaml:"source_model_name"`
	SourceModelVersion  string `json:"source_model_version" yaml:"source_model_version"`
	TargetModelName     string `json:"target_model_name" yaml:"target_model_name"`
	TargetModelVersion  string `json:"target_model_version" yaml:"target_model_version"`
	InteractionType     string `json:"interaction_type" yaml:"interaction_type"`
	Environment         string `json:"environment" yaml:"environment"`
	Timestamp           string `json:"timestamp" yaml:"timestamp"`
	MetadataSnapshotRef string `json:"metadata_snapshot_ref" yaml:"metadata_snapshot_ref"`
}

func (m InteractionMetadata) Validate() error {
	var issues []string
	add := func(msg string) {
		issues = append(issues, msg)
	}

	if m.SourceModelName == "" {
		add("interaction.source_model_name: required")
	}
	if m.SourceModelVersion == "" {
		add("interaction.source_model_version: required")
	}
	if m.TargetModelName == "" {
		add("interaction.target_model_name: required")
	}
	if m.TargetModelVersion == "" {
		add("interaction.target_model_version: required")
	}
	if m.InteractionType == "" {
		add("interaction.interaction_type: required")
	}
	if m.Environment == "" {
		add("interaction.environment: required")
	} else if !isValidEnvironment(m.Environment) {
		add("interaction.environment: must be dev, staging, or production")
	}
	if m.Timestamp == "" {
		add("interaction.timestamp: required")
	} else if _, err := time.Parse(time.RFC3339, m.Timestamp); err != nil {
		add("interaction.timestamp: must be RFC3339")
	}
	if m.MetadataSnapshotRef == "" {
		add("interaction.metadata_snapshot_ref: required")
	}

	if len(issues) > 0 {
		return &ValidationError{Issues: issues}
	}
	return nil
}

// InteractionMetadataCollector assembles observed interaction metadata.
type InteractionMetadataCollector interface {
	CollectInteraction(action ActionContext) (InteractionMetadata, error)
}

// StaticModelMetadataCollector returns a fixed ModelMetadata payload.
type StaticModelMetadataCollector struct {
	Metadata ModelMetadata
}

func (c StaticModelMetadataCollector) Collect(action ActionContext) (ModelMetadata, error) {
	return c.Metadata, nil
}

// StaticInteractionMetadataCollector returns a fixed InteractionMetadata payload.
type StaticInteractionMetadataCollector struct {
	Metadata InteractionMetadata
}

func (c StaticInteractionMetadataCollector) CollectInteraction(action ActionContext) (InteractionMetadata, error) {
	return c.Metadata, nil
}
