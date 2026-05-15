package session

import (
	"os"
	"testing"
	"time"

	"github.com/raydraw/ergate/internal/llm"
)

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}

	sess := &Session{
		ID:        "test_session",
		CreatedAt: time.Now(),
		Model:     "claude-test",
		Messages: []llm.Message{
			{Role: "user", Content: []llm.ContentBlock{{Type: "text", Text: "hello"}}},
			{Role: "assistant", Content: []llm.ContentBlock{{Type: "text", Text: "hi!"}}},
		},
		Usage: llm.Usage{InputTokens: 5, OutputTokens: 3},
	}

	if err := store.Save(sess); err != nil {
		t.Fatal(err)
	}

	loaded, err := store.Load("test_session")
	if err != nil {
		t.Fatal(err)
	}

	if loaded.Model != "claude-test" {
		t.Errorf("model: got %q, want %q", loaded.Model, "claude-test")
	}
	if len(loaded.Messages) != 2 {
		t.Errorf("messages: got %d, want 2", len(loaded.Messages))
	}
	if loaded.Usage.InputTokens != 5 {
		t.Errorf("input tokens: got %d, want 5", loaded.Usage.InputTokens)
	}
}

func TestListSessions(t *testing.T) {
	dir := t.TempDir()
	store, _ := NewStore(dir)

	store.Save(&Session{ID: "b", CreatedAt: time.Now().Add(-1 * time.Hour)})
	store.Save(&Session{ID: "a", CreatedAt: time.Now()})

	ids, err := store.List()
	if err != nil {
		t.Fatal(err)
	}

	if len(ids) != 2 {
		t.Errorf("expected 2 sessions, got %d", len(ids))
	}
	// Most recent first
	if ids[0] != "a" {
		t.Errorf("expected 'a' first, got %q", ids[0])
	}
}

func TestDeleteSession(t *testing.T) {
	dir := t.TempDir()
	store, _ := NewStore(dir)

	store.Save(&Session{ID: "to_delete"})
	store.Delete("to_delete")

	_, err := store.Load("to_delete")
	if err == nil {
		t.Error("expected error loading deleted session")
	}
	if !os.IsNotExist(err) {
		t.Logf("error type: %v", err)
	}
}

func TestLatestSession(t *testing.T) {
	dir := t.TempDir()
	store, _ := NewStore(dir)

	// No sessions
	sess, err := store.Latest()
	if err != nil {
		t.Fatal(err)
	}
	if sess != nil {
		t.Error("expected nil for empty store")
	}

	store.Save(&Session{ID: "latest", CreatedAt: time.Now()})
	sess, err = store.Latest()
	if err != nil {
		t.Fatal(err)
	}
	if sess.ID != "latest" {
		t.Errorf("expected 'latest', got %q", sess.ID)
	}
}

func TestEngineExportImport(t *testing.T) {
	// Test that engine can export and import session data
	dir := t.TempDir()
	store, _ := NewStore(dir)

	sess := &Session{
		ID:    "engine_test",
		Model: "test",
		Messages: []llm.Message{
			{Role: "user", Content: []llm.ContentBlock{{Type: "text", Text: "test"}}},
		},
	}

	store.Save(sess)
	loaded, _ := store.Load("engine_test")
	if len(loaded.Messages) != 1 {
		t.Errorf("messages: got %d", len(loaded.Messages))
	}
}
