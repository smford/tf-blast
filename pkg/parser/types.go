package parser

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Plan represents the top-level Terraform / OpenTofu JSON execution plan.
type Plan struct {
	FormatVersion      string           `json:"format_version"`
	TerraformVersion   string           `json:"terraform_version,omitempty"`
	Variables          map[string]any   `json:"variables,omitempty"`
	PlannedValues      *PlannedValues   `json:"planned_values,omitempty"`
	ResourceChanges    []ResourceChange `json:"resource_changes,omitempty"`
	ResourceDrift      []ResourceDrift  `json:"resource_drift,omitempty"`
	PriorState         *PriorState      `json:"prior_state,omitempty"`
	Configuration      *Configuration   `json:"configuration,omitempty"`
	RelevantAttributes []any            `json:"relevant_attributes,omitempty"`
}

// ResourceDrift describes out-of-band changes detected between actual cloud state and statefile.
type ResourceDrift struct {
	Address       string `json:"address"`
	ModuleAddress string `json:"module_address,omitempty"`
	Mode          string `json:"mode"`
	Type          string `json:"type"`
	Name          string `json:"name"`
	ProviderName  string `json:"provider_name"`
	Change        Change `json:"change"`
}

// ResourceChange describes a planned modification for a single resource.
type ResourceChange struct {
	Address         string `json:"address"`
	PreviousAddress string `json:"previous_address,omitempty"`
	ModuleAddress   string `json:"module_address,omitempty"`
	Mode            string `json:"mode"`
	Type            string `json:"type"`
	Name            string `json:"name"`
	Index           any    `json:"index,omitempty"`
	ProviderName    string `json:"provider_name"`
	Change          Change `json:"change"`
	ActionReason    string `json:"action_reason,omitempty"`
	PlanSource      string `json:"plan_source,omitempty"`
}

// Change encapsulates the actions and diff states.
type Change struct {
	Actions         []string       `json:"actions"`
	Before          map[string]any `json:"before,omitempty"`
	After           map[string]any `json:"after,omitempty"`
	AfterUnknown    map[string]any `json:"after_unknown,omitempty"`
	BeforeSensitive any            `json:"before_sensitive,omitempty"`
	AfterSensitive  any            `json:"after_sensitive,omitempty"`
	ReplacePaths    [][]any        `json:"replace_paths,omitempty"`
	ActionReason    string         `json:"action_reason,omitempty"`
}

// Configuration describes the Terraform HCL configuration model.
type Configuration struct {
	ProviderConfig map[string]any `json:"provider_config,omitempty"`
	RootModule     ConfigModule   `json:"root_module"`
}

// ConfigModule represents a module block within the configuration graph.
type ConfigModule struct {
	Resources   []ConfigResource      `json:"resources,omitempty"`
	ModuleCalls map[string]ModuleCall `json:"module_calls,omitempty"`
}

// ConfigResource is a resource defined in configuration.
type ConfigResource struct {
	Address           string                `json:"address"`
	Mode              string                `json:"mode"`
	Type              string                `json:"type"`
	Name              string                `json:"name"`
	ProviderConfigKey string                `json:"provider_config_key,omitempty"`
	Expressions       map[string]Expression `json:"expressions,omitempty"`
	DependsOn         []string              `json:"depends_on,omitempty"`
	CountExpression   *Expression           `json:"count_expression,omitempty"`
	ForEachExpression *Expression           `json:"for_each_expression,omitempty"`
}

// ModuleCall is a call to a child module.
type ModuleCall struct {
	Source      string                `json:"source,omitempty"`
	Expressions map[string]Expression `json:"expressions,omitempty"`
	DependsOn   []string              `json:"depends_on,omitempty"`
	Module      *ConfigModule         `json:"module,omitempty"`
}

// Expression represents an HCL expression in configuration.
type Expression struct {
	References    []string `json:"references,omitempty"`
	ConstantValue any      `json:"constant_value,omitempty"`
}

// PlannedValues contains the planned values tree.
type PlannedValues struct {
	RootModule StateModule `json:"root_module"`
}

// PriorState holds prior state information.
type PriorState struct {
	FormatVersion    string          `json:"format_version,omitempty"`
	TerraformVersion string          `json:"terraform_version,omitempty"`
	Values           *StateContainer `json:"values,omitempty"`
	RootModule       *StateModule    `json:"root_module,omitempty"`
}

// StateContainer is a wrapper around root_module in some state schemas.
type StateContainer struct {
	RootModule StateModule `json:"root_module"`
}

// StateModule represents a module in prior state or planned values.
type StateModule struct {
	Address      string          `json:"address,omitempty"`
	Resources    []StateResource `json:"resources,omitempty"`
	ChildModules []StateModule   `json:"child_modules,omitempty"`
}

// StateResource represents a resource in prior state or planned values.
type StateResource struct {
	Address      string         `json:"address"`
	Mode         string         `json:"mode"`
	Type         string         `json:"type"`
	Name         string         `json:"name"`
	Index        any            `json:"index,omitempty"`
	ProviderName string         `json:"provider_name"`
	Values       map[string]any `json:"values,omitempty"`
}

// UnmarshalJSON custom handler for Change to tolerate before/after being non-maps (e.g., primitives).
func (c *Change) UnmarshalJSON(data []byte) error {
	type Alias Change
	aux := &struct {
		BeforeRaw json.RawMessage `json:"before"`
		AfterRaw  json.RawMessage `json:"after"`
		*Alias
	}{
		Alias: (*Alias)(c),
	}
	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}
	if len(aux.BeforeRaw) > 0 && string(aux.BeforeRaw) != "null" {
		var m map[string]any
		if err := json.Unmarshal(aux.BeforeRaw, &m); err == nil {
			c.Before = m
		}
	}
	if len(aux.AfterRaw) > 0 && string(aux.AfterRaw) != "null" {
		var m map[string]any
		if err := json.Unmarshal(aux.AfterRaw, &m); err == nil {
			c.After = m
		}
	}
	return nil
}

// FormatReplacePath converts a replace_path entry (e.g., ["ingress", 0, "cidr_blocks"]) to string.
func FormatReplacePath(path []any) string {
	var result string
	for i, seg := range path {
		switch v := seg.(type) {
		case string:
			if i == 0 {
				result = v
			} else {
				result += "." + v
			}
		case float64:
			result += fmt.Sprintf("[%d]", int(v))
		case int:
			result += fmt.Sprintf("[%d]", v)
		default:
			if i == 0 {
				result = fmt.Sprintf("%v", v)
			} else {
				result += fmt.Sprintf(".%v", v)
			}
		}
	}
	return result
}

// IsFieldSensitive checks if a specific field is marked sensitive in Terraform's sensitive maps,
// or matches common sensitive keyword patterns.
func IsFieldSensitive(fieldName string, sensitiveMap any) bool {
	lowerName := strings.ToLower(fieldName)
	if strings.Contains(lowerName, "password") ||
		strings.Contains(lowerName, "secret") ||
		strings.Contains(lowerName, "token") ||
		strings.Contains(lowerName, "private_key") ||
		strings.Contains(lowerName, "master_password") ||
		strings.Contains(lowerName, "api_key") {
		return true
	}

	if sensitiveMap == nil {
		return false
	}

	switch sm := sensitiveMap.(type) {
	case bool:
		return sm
	case map[string]any:
		if val, exists := sm[fieldName]; exists {
			if b, isBool := val.(bool); isBool {
				return b
			}
			return true
		}
	}
	return false
}

// SanitizeValue returns "(sensitive value redacted)" if the field is sensitive, otherwise returns the value formatted.
func SanitizeValue(fieldName string, value any, beforeSensitive, afterSensitive any) string {
	if IsFieldSensitive(fieldName, beforeSensitive) || IsFieldSensitive(fieldName, afterSensitive) {
		return "(sensitive value redacted)"
	}
	return fmt.Sprintf("%v", value)
}
