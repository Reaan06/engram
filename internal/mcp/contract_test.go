package mcp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

type toolContract struct {
	Properties map[string]string `json:"properties"`
	Required   []string          `json:"required"`
}

// TestRuntimeToolContract compares the protocol-visible registry against the
// repository-owned compatibility baseline. Descriptions and ordering are
// deliberately excluded: they are not compatibility requirements.
func TestRuntimeToolContract(t *testing.T) {
	s := newMCPTestStore(t)
	actual := make(map[string]toolContract)
	for name, registered := range NewServer(s).ListTools() {
		properties := make(map[string]string, len(registered.Tool.InputSchema.Properties))
		for property, raw := range registered.Tool.InputSchema.Properties {
			object, ok := raw.(map[string]any)
			if !ok {
				t.Fatalf("tool %q property %q has unexpected schema value %T", name, property, raw)
			}
			typ, ok := object["type"].(string)
			if !ok {
				t.Fatalf("tool %q property %q has no scalar type", name, property)
			}
			properties[property] = typ
		}
		required := append([]string(nil), registered.Tool.InputSchema.Required...)
		sort.Strings(required)
		actual[name] = toolContract{Properties: properties, Required: required}
	}

	data, err := os.ReadFile(filepath.Join("testdata", "mcp-tools-baseline-v1.json"))
	if err != nil {
		t.Fatal(err)
	}
	var expected map[string]toolContract
	if err := json.Unmarshal(data, &expected); err != nil {
		t.Fatal(err)
	}
	missingTools, unexpectedTools := contractSetDiff(sortedContractNames(expected), sortedContractNames(actual))
	if len(missingTools) > 0 {
		t.Errorf("tool inventory drift: missing tools: %s", strings.Join(missingTools, ", "))
	}
	if len(unexpectedTools) > 0 {
		t.Errorf("tool inventory drift: unexpected tools: %s", strings.Join(unexpectedTools, ", "))
	}
	for _, name := range sortedContractNames(expected) {
		want := expected[name]
		got, exists := actual[name]
		if !exists {
			continue
		}
		missingProperties, unexpectedProperties := contractSetDiff(sortedPropertyNames(want.Properties), sortedPropertyNames(got.Properties))
		if len(missingProperties) > 0 {
			t.Errorf("tool %q property drift: missing properties: %s", name, strings.Join(missingProperties, ", "))
		}
		if len(unexpectedProperties) > 0 {
			t.Errorf("tool %q property drift: unexpected properties: %s", name, strings.Join(unexpectedProperties, ", "))
		}
		for _, property := range sortedPropertyNames(want.Properties) {
			if gotType, exists := got.Properties[property]; exists && gotType != want.Properties[property] {
				t.Errorf("tool %q property %q type drift: got %q, want %q", name, property, gotType, want.Properties[property])
			}
		}
		missingRequired, unexpectedRequired := contractSetDiff(want.Required, got.Required)
		if len(missingRequired) > 0 {
			t.Errorf("tool %q required-field drift: missing fields: %s", name, strings.Join(missingRequired, ", "))
		}
		if len(unexpectedRequired) > 0 {
			t.Errorf("tool %q required-field drift: unexpected fields: %s", name, strings.Join(unexpectedRequired, ", "))
		}
	}
}

func sortedPropertyNames(properties map[string]string) []string {
	names := make([]string, 0, len(properties))
	for name := range properties {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func contractSetDiff(want, got []string) (missing, unexpected []string) {
	gotSet := make(map[string]struct{}, len(got))
	for _, value := range got {
		gotSet[value] = struct{}{}
	}
	for _, value := range want {
		if _, exists := gotSet[value]; !exists {
			missing = append(missing, value)
		}
	}
	wantSet := make(map[string]struct{}, len(want))
	for _, value := range want {
		wantSet[value] = struct{}{}
	}
	for _, value := range got {
		if _, exists := wantSet[value]; !exists {
			unexpected = append(unexpected, value)
		}
	}
	sort.Strings(missing)
	sort.Strings(unexpected)
	return missing, unexpected
}

func sortedContractNames(contracts map[string]toolContract) []string {
	names := make([]string, 0, len(contracts))
	for name := range contracts {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
