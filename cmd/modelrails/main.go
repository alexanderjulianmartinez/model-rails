package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/alexanderjulianmartinez/model-rails/internal/engine"
	"github.com/alexanderjulianmartinez/model-rails/internal/spec"
	"gopkg.in/yaml.v3"
)

const (
	exitAllow = 0
	exitBlock = 1
	exitError = 2
)

type outputPayload struct {
	Decision    string             `json:"decision"`
	Violations  []engine.Violation `json:"violations"`
	Explanation string             `json:"explanation"`
}

func main() {
	var (
		specPath     string
		actionPath   string
		metadataPath string
		outputMode   string
		strict       bool
		failOnWarn   bool
	)

	flag.StringVar(&specPath, "spec", "", "Path to invariant spec (YAML/JSON)")
	flag.StringVar(&actionPath, "action", "", "Path to action context (YAML/JSON)")
	flag.StringVar(&metadataPath, "metadata", "", "Path to model metadata (YAML/JSON)")
	flag.StringVar(&outputMode, "output", "human", "Output format: human|json")
	flag.BoolVar(&strict, "strict", false, "Treat WARN as BLOCK")
	flag.BoolVar(&failOnWarn, "fail-on-warn", false, "Exit 1 when decision is WARN")
	flag.Parse()

	if specPath == "" || actionPath == "" || metadataPath == "" {
		fatal(exitError, "missing required flags: --spec, --action, --metadata")
	}

	if outputMode != "human" && outputMode != "json" {
		fatal(exitError, "invalid --output value: must be human or json")
	}

	var specDoc spec.InvariantSpec
	if err := decodeFile(specPath, &specDoc); err != nil {
		fatal(exitError, fmt.Sprintf("failed to load spec: %v", err))
	}

	var action engine.ActionContext
	if err := decodeFile(actionPath, &action); err != nil {
		fatal(exitError, fmt.Sprintf("failed to load action: %v", err))
	}

	var metadata engine.ModelMetadata
	if err := decodeFile(metadataPath, &metadata); err != nil {
		fatal(exitError, fmt.Sprintf("failed to load metadata: %v", err))
	}

	result, err := engine.Evaluate(specDoc, action, metadata)
	if err != nil {
		fatal(exitError, err.Error())
	}

	if strict && result.Decision == engine.DecisionWarn {
		result.Decision = engine.DecisionBlock
		result.Explanation = "STRICT MODE: WARN treated as BLOCK\n" + result.Explanation
	}

	render(result, outputMode)

	switch result.Decision {
	case engine.DecisionAllow:
		exit(exitAllow)
	case engine.DecisionWarn:
		if failOnWarn {
			exit(exitBlock)
		}
		exit(exitAllow)
	case engine.DecisionBlock:
		exit(exitBlock)
	default:
		exit(exitError)
	}
}

func render(result engine.DecisionResult, mode string) {
	if mode == "json" {
		payload := outputPayload{
			Decision:    result.Decision,
			Violations:  result.Violations,
			Explanation: result.Explanation,
		}
		data, err := json.MarshalIndent(payload, "", "  ")
		if err != nil {
			fatal(exitError, fmt.Sprintf("failed to render json: %v", err))
		}
		fmt.Println(string(data))
		return
	}

	fmt.Println(engine.BuildDecisionOutput(result, engine.CIFormatter{}))
}

func decodeFile(path string, out interface{}) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" {
		return errors.New("file is empty")
	}

	if strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[") {
		return json.Unmarshal([]byte(trimmed), out)
	}

	return yaml.Unmarshal([]byte(trimmed), out)
}

func fatal(code int, message string) {
	fmt.Fprintln(os.Stderr, message)
	exit(code)
}

func exit(code int) {
	os.Exit(code)
}
