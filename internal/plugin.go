// Package internal implements the workflow-plugin-observability plugin.
package internal

import (
	"fmt"

	sdk "github.com/GoCodeAlone/workflow/plugin/external/sdk"
)

// Version is set at build time via -ldflags
// "-X github.com/GoCodeAlone/workflow-plugin-observability/internal.Version=X.Y.Z".
// Default is a bare semver so plugin loaders that validate semver accept
// unreleased dev builds; goreleaser overrides with the real release tag.
var Version = "0.0.0"

// observabilityPlugin implements sdk.PluginProvider and optionally
// sdk.ModuleProvider, sdk.StepProvider, sdk.TriggerProvider, etc.
type observabilityPlugin struct{}

// NewPlugin returns a new plugin instance. main.go calls sdk.Serve(NewPlugin()).
func NewPlugin() sdk.PluginProvider {
	return &observabilityPlugin{}
}

// Manifest returns the plugin metadata used by the workflow engine for
// discovery and capability negotiation.
func (p *observabilityPlugin) Manifest() sdk.PluginManifest {
	return sdk.PluginManifest{
		Name:        "workflow-plugin-observability",
		Version:     Version,
		Author:      "GoCodeAlone",
		Description: "observability plugin for the workflow engine",
	}
}

// ModuleTypes returns the module type names this plugin provides.
// Remove this method if the plugin does not provide any modules.
func (p *observabilityPlugin) ModuleTypes() []string {
	return []string{
		"observability.collector",
		"observability.telemetry",
	}
}

// CreateModule creates a module instance of the given type.
// Remove this method if the plugin does not provide any modules.
func (p *observabilityPlugin) CreateModule(typeName, name string, config map[string]any) (sdk.ModuleInstance, error) {
	switch typeName {
	case "observability.collector":
		return newCollectorModule(name, config)
	case "observability.telemetry":
		return newTelemetryModule(name, config)
	default:
		return nil, fmt.Errorf("observability: unknown module type %q", typeName)
	}
}

// StepTypes returns the step type names this plugin provides.
// Remove this method if the plugin does not provide any steps.
func (p *observabilityPlugin) StepTypes() []string {
	return []string{
		// "step.example_action",
	}
}

// CreateStep creates a step instance of the given type.
// Remove this method if the plugin does not provide any steps.
func (p *observabilityPlugin) CreateStep(typeName, name string, config map[string]any) (sdk.StepInstance, error) {
	switch typeName {
	// case "step.example_action":
	//     return newExampleStep(name, config), nil
	default:
		return nil, fmt.Errorf("observability: unknown step type %q", typeName)
	}
}
