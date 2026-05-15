package mcp

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/raydraw/ergate/internal/llm"
)

// Client is an MCP client that connects to an MCP server.
type Client struct {
	transport Transport
	info      ServerInfo
	tools     []Tool

	mu sync.RWMutex
}

// NewClient creates an MCP client and initializes the connection.
func NewClient(transport Transport) (*Client, error) {
	c := &Client{transport: transport}

	if err := c.initialize(); err != nil {
		transport.Close()
		return nil, fmt.Errorf("initialize: %w", err)
	}

	if err := c.discoverTools(); err != nil {
		transport.Close()
		return nil, fmt.Errorf("discover tools: %w", err)
	}

	return c, nil
}

func (c *Client) initialize() error {
	params := InitializeParams{
		ProtocolVersion: ProtocolVersion,
		Capabilities: Capabilities{
			Roots: &RootsCapability{ListChanged: true},
		},
		ClientInfo: ClientInfo{
			Name:    "ergate",
			Version: "0.1.0",
		},
	}

	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return err
	}

	resp, err := c.transport.Send(&Request{
		Method: MethodInitialize,
		Params: json.RawMessage(paramsJSON),
	})
	if err != nil {
		return err
	}

	var result InitializeResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return fmt.Errorf("unmarshal initialize result: %w", err)
	}

	c.info = result.ServerInfo
	return nil
}

func (c *Client) discoverTools() error {
	resp, err := c.transport.Send(&Request{
		Method: MethodToolsList,
	})
	if err != nil {
		return err
	}

	var result ListToolsResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return fmt.Errorf("unmarshal tools list: %w", err)
	}

	c.mu.Lock()
	c.tools = result.Tools
	c.mu.Unlock()

	return nil
}

// Tools returns the server's tools.
func (c *Client) Tools() []Tool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	result := make([]Tool, len(c.tools))
	copy(result, c.tools)
	return result
}

// ToolConfigs converts MCP tools to LLM tool configs.
func (c *Client) ToolConfigs() []llm.ToolConfig {
	c.mu.RLock()
	defer c.mu.RUnlock()

	configs := make([]llm.ToolConfig, 0, len(c.tools))
	for _, t := range c.tools {
		configs = append(configs, llm.ToolConfig{
			Name:        "mcp__" + c.info.Name + "__" + t.Name,
			Description: t.Description,
			InputSchema: t.InputSchema,
		})
	}
	return configs
}

// CallTool invokes a tool on the MCP server.
func (c *Client) CallTool(name string, args json.RawMessage) (*CallToolResult, error) {
	params := CallToolParams{
		Name:      name,
		Arguments: args,
	}

	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}

	resp, err := c.transport.Send(&Request{
		Method: MethodToolsCall,
		Params: json.RawMessage(paramsJSON),
	})
	if err != nil {
		return nil, err
	}

	var result CallToolResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("unmarshal tool result: %w", err)
	}

	return &result, nil
}

// ServerName returns the connected server's name.
func (c *Client) ServerName() string {
	return c.info.Name
}

// Close shuts down the MCP connection.
func (c *Client) Close() error {
	return c.transport.Close()
}
