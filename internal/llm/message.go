package llm

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"time"
)

// MessageType is the discriminated union type for messages.
type MessageType string

const (
	MsgUser       MessageType = "user"
	MsgAssistant  MessageType = "assistant"
	MsgSystem     MessageType = "system"
	MsgProgress   MessageType = "progress"
	MsgAttachment MessageType = "attachment"
)

// SystemSubtype discriminates system message variants.
type SystemSubtype string

const (
	SysInformational       SystemSubtype = "informational"
	SysCompactBoundary     SystemSubtype = "compact_boundary"
	SysMicrocompactBoundary SystemSubtype = "microcompact_boundary"
	SysAPIError            SystemSubtype = "api_error"
	SysPermissionRetry     SystemSubtype = "permission_retry"
	SysBridgeStatus        SystemSubtype = "bridge_status"
	SysTurnDuration        SystemSubtype = "turn_duration"
	SysMemorySaved         SystemSubtype = "memory_saved"
)

// SystemLevel indicates severity.
type SystemLevel string

const (
	LevelInfo    SystemLevel = "info"
	LevelWarning SystemLevel = "warning"
	LevelError   SystemLevel = "error"
)

func newUUID() string {
	b := make([]byte, 16)
	rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// NewUserMessage creates a user message with text content.
func NewUserMessage(text string) Message {
	return Message{
		Type: MsgUser,
		UUID: newUUID(),
		Timestamp: time.Now(),
		Role: "user",
		Content: []ContentBlock{{Type: "text", Text: text}},
	}
}

// NewAssistantMessage creates an assistant message from API response.
func NewAssistantMessage(id, model, stopReason string, blocks []ContentBlock) Message {
	return Message{
		Type: MsgAssistant,
		UUID: newUUID(),
		Timestamp: time.Now(),
		Role: "assistant",
		Content: blocks,
		MessageID: id,
		Model: model,
		StopReason: stopReason,
	}
}

// NewSystemMessage creates a system message for a given subtype.
func NewSystemMessage(subtype SystemSubtype, content string, level SystemLevel) Message {
	return Message{
		Type: MsgSystem,
		UUID: newUUID(),
		Timestamp: time.Now(),
		Role: "system",
		Subtype: string(subtype),
		Level: string(level),
		Content: []ContentBlock{{Type: "text", Text: content}},
	}
}

// NewCompactBoundary creates a boundary marker for compaction.
func NewCompactBoundary(trigger string, preTokens int, summary string) Message {
	m := NewSystemMessage(SysCompactBoundary, summary, LevelInfo)
	m.IsMeta = true
	// Compact metadata stored in the text block's Text field for serialization.
	return m
}

// NewToolResultMessage creates a user-tool-result message.
func NewToolResultMessage(toolUseID, content string, isError bool) Message {
	encoded, _ := json.Marshal(content)
	return Message{
		Type: MsgUser,
		UUID: newUUID(),
		Timestamp: time.Now(),
		Role: "user",
		Content: []ContentBlock{{
			Type:      "tool_result",
			ToolUseID: toolUseID,
			Content:   json.RawMessage(encoded),
			IsError:   isError,
		}},
	}
}

// IsSystem returns true for system messages.
func (m *Message) IsSystem() bool { return m.Type == MsgSystem }

// IsAssistant returns true for assistant messages.
func (m *Message) IsAssistant() bool { return m.Type == MsgAssistant }

// IsUser returns true for user messages.
func (m *Message) IsUser() bool { return m.Type == MsgUser }
