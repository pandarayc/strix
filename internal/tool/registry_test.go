package tool

import (
	"context"
	"encoding/json"
	"testing"
)

type testTool struct {
	BaseTool
}

func newTestTool(name string) *testTool {
	return &testTool{
		BaseTool: NewBaseTool(name, "test tool", json.RawMessage(`{"type":"object"}`)),
	}
}

func (t *testTool) Execute(ctx context.Context, input json.RawMessage, exec *ExecContext) (*ToolResult, error) {
	return &ToolResult{Success: true, Content: "ok"}, nil
}

func TestRegistryRegisterAndGet(t *testing.T) {
	reg := NewRegistry()

	tool := newTestTool("test_tool")
	if err := reg.Register(tool); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	got, ok := reg.Get("test_tool")
	if !ok {
		t.Fatal("Get returned false for registered tool")
	}
	if got.Name() != "test_tool" {
		t.Errorf("Expected name 'test_tool', got %q", got.Name())
	}
}

func TestRegistryDuplicateRegister(t *testing.T) {
	reg := NewRegistry()

	reg.Register(newTestTool("test_tool"))
	err := reg.Register(newTestTool("test_tool"))
	if err == nil {
		t.Error("Expected error for duplicate registration")
	}
}

func TestRegistryList(t *testing.T) {
	reg := NewRegistry()
	reg.Register(newTestTool("tool_a"))
	reg.Register(newTestTool("tool_b"))

	list := reg.List()
	if len(list) != 2 {
		t.Errorf("Expected 2 tools, got %d", len(list))
	}
}

func TestRegistryToolConfigs(t *testing.T) {
	reg := NewRegistry()
	reg.Register(newTestTool("enabled_tool"))

	disabled := newTestTool("disabled_tool")
	disabled.enabled = false
	reg.Register(disabled)

	configs := reg.ToolConfigs()
	if len(configs) != 1 {
		t.Errorf("Expected 1 enabled tool config, got %d", len(configs))
	}
	if configs[0].Name != "enabled_tool" {
		t.Errorf("Expected 'enabled_tool', got %q", configs[0].Name)
	}
}
