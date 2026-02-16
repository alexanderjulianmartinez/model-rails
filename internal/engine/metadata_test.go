package engine

import (
	"strings"
	"testing"
)

func TestInteractionMetadataValidate(t *testing.T) {
	valid := InteractionMetadata{
		SourceModelName:     "rec-model",
		SourceModelVersion:  "v1.2.3",
		TargetModelName:     "embedding-service",
		TargetModelVersion:  "v2.0.0",
		InteractionType:     "call",
		Environment:         EnvDev,
		Timestamp:           "2026-02-16T12:00:00Z",
		MetadataSnapshotRef: "snapshot://interaction-1",
	}

	if err := valid.Validate(); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	invalid := InteractionMetadata{}
	err := invalid.Validate()
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	checks := []string{
		"interaction.source_model_name",
		"interaction.source_model_version",
		"interaction.target_model_name",
		"interaction.target_model_version",
		"interaction.interaction_type",
		"interaction.environment",
		"interaction.timestamp",
		"interaction.metadata_snapshot_ref",
	}

	for _, needle := range checks {
		if !strings.Contains(err.Error(), needle) {
			t.Fatalf("expected error to contain %s, got %v", needle, err)
		}
	}
}

func TestStaticModelMetadataCollector(t *testing.T) {
	metadata := validMetadata()
	collector := StaticModelMetadataCollector{Metadata: metadata}

	got, err := collector.Collect(validAction())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got.ModelName != metadata.ModelName {
		t.Fatalf("expected model name %s, got %s", metadata.ModelName, got.ModelName)
	}
}

func TestStaticInteractionMetadataCollector(t *testing.T) {
	metadata := InteractionMetadata{
		SourceModelName:     "rec-model",
		SourceModelVersion:  "v1.2.3",
		TargetModelName:     "embedding-service",
		TargetModelVersion:  "v2.0.0",
		InteractionType:     "call",
		Environment:         EnvDev,
		Timestamp:           "2026-02-16T12:00:00Z",
		MetadataSnapshotRef: "snapshot://interaction-1",
	}
	collector := StaticInteractionMetadataCollector{Metadata: metadata}

	got, err := collector.CollectInteraction(validAction())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got.TargetModelName != metadata.TargetModelName {
		t.Fatalf("expected target model %s, got %s", metadata.TargetModelName, got.TargetModelName)
	}
}
