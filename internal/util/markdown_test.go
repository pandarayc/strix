package util

import (
	"strings"
	"testing"
)

func TestBold(t *testing.T) {
	result := RenderMarkdown("hello **world** here", 80)
	if !strings.Contains(result, "world") {
		t.Error("expected bold text in output")
	}
}

func TestInlineCode(t *testing.T) {
	result := RenderMarkdown("use `fmt.Println()` to print", 80)
	if !strings.Contains(result, "fmt.Println()") {
		t.Error("expected inline code in output")
	}
}

func TestHeader(t *testing.T) {
	result := RenderMarkdown("# Title\n\ncontent", 80)
	if !strings.Contains(result, "Title") {
		t.Error("expected header text in output")
	}
}

func TestCodeBlock(t *testing.T) {
	result := RenderMarkdown("```go\nfunc main() {}\n```", 80)
	if !strings.Contains(result, "go") && !strings.Contains(result, "func main") {
		t.Error("expected code block content in output")
	}
}

func TestLink(t *testing.T) {
	result := RenderMarkdown("[click here](https://example.com)", 80)
	// Link text should be preserved, URL might be dropped
	if !strings.Contains(result, "click here") {
		t.Error("expected link text in output")
	}
}

func TestUnorderedList(t *testing.T) {
	result := RenderMarkdown("- item one\n- item two", 80)
	if !strings.Contains(result, "item one") || !strings.Contains(result, "item two") {
		t.Error("expected list items in output")
	}
}

func TestOrderedList(t *testing.T) {
	result := RenderMarkdown("1. first\n2. second", 80)
	if !strings.Contains(result, "first") || !strings.Contains(result, "second") {
		t.Error("expected ordered list items in output")
	}
}

func TestBlockquote(t *testing.T) {
	result := RenderMarkdown("> This is a quote", 80)
	if !strings.Contains(result, "This is a quote") {
		t.Error("expected quote text in output")
	}
}

func TestTable(t *testing.T) {
	result := RenderMarkdown("| Name | Value |\n|------|-------|\n| foo  | bar   |", 80)
	if !strings.Contains(result, "foo") || !strings.Contains(result, "bar") {
		t.Error("expected table cells in output")
	}
}

func TestHorizontalRule(t *testing.T) {
	result := RenderMarkdown("before\n---\nafter", 80)
	if !strings.Contains(result, "before") || !strings.Contains(result, "after") {
		t.Error("expected content before and after hr")
	}
}

func TestItalic(t *testing.T) {
	result := RenderMarkdown("hello *world* here", 80)
	if !strings.Contains(result, "world") {
		t.Error("expected italic text in output")
	}
}

func TestEmptyInput(t *testing.T) {
	result := RenderMarkdown("", 80)
	if result == "" {
		t.Error("expected empty string for empty input")
	}
}
