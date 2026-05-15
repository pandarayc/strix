package mcp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
)

// Transport is the communication layer for MCP.
type Transport interface {
	// Send writes a request and returns the response.
	Send(req *Request) (*Response, error)
	// Close shuts down the transport.
	Close() error
}

// StdioTransport communicates with an MCP server via stdin/stdout.
type StdioTransport struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Scanner
	id     int64
}

// NewStdioTransport spawns an MCP server process and connects to its stdio.
func NewStdioTransport(command string, args ...string) (*StdioTransport, error) {
	cmd := exec.Command(command, args...)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start server: %w", err)
	}

	return &StdioTransport{
		cmd:    cmd,
		stdin:  stdin,
		stdout: bufio.NewScanner(stdout),
	}, nil
}

// Send writes a request and reads the response.
func (t *StdioTransport) Send(req *Request) (*Response, error) {
	t.id++
	req.ID = t.id
	req.JSONRPC = JSONRPCVersion

	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	// Write request followed by newline
	if _, err := fmt.Fprintf(t.stdin, "%s\n", data); err != nil {
		return nil, fmt.Errorf("write request: %w", err)
	}

	// Read response
	if !t.stdout.Scan() {
		if err := t.stdout.Err(); err != nil {
			return nil, fmt.Errorf("read response: %w", err)
		}
		return nil, fmt.Errorf("server closed connection")
	}

	var resp Response
	if err := json.Unmarshal(t.stdout.Bytes(), &resp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	if resp.Error != nil {
		return &resp, fmt.Errorf("rpc error %d: %s", resp.Error.Code, resp.Error.Message)
	}

	return &resp, nil
}

// Close terminates the server process.
func (t *StdioTransport) Close() error {
	if t.stdin != nil {
		t.stdin.Close()
	}
	if t.cmd != nil && t.cmd.Process != nil {
		return t.cmd.Process.Kill()
	}
	return nil
}

// SSETransport communicates with an MCP server via Server-Sent Events.
type SSETransport struct {
	baseURL string
	id      int64
	client  *http.Client
}

// NewSSETransport creates an SSE-based MCP transport.
func NewSSETransport(baseURL string) *SSETransport {
	return &SSETransport{
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  &http.Client{},
	}
}

// Send posts a JSON-RPC request to the SSE server's /message endpoint.
func (t *SSETransport) Send(req *Request) (*Response, error) {
	t.id++
	req.ID = t.id
	req.JSONRPC = JSONRPCVersion

	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", t.baseURL+"/message", bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")

	resp, err := t.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("sse request: %w", err)
	}
	defer resp.Body.Close()

	// SSE response: read the event stream for the JSON-RPC response
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			var rpcResp Response
			if err := json.Unmarshal([]byte(data), &rpcResp); err != nil {
				continue
			}
			if rpcResp.Error != nil {
				return &rpcResp, fmt.Errorf("rpc error %d: %s", rpcResp.Error.Code, rpcResp.Error.Message)
			}
			return &rpcResp, nil
		}
	}
	return nil, fmt.Errorf("no response from SSE server")
}

func (t *SSETransport) Close() error {
	t.client.CloseIdleConnections()
	return nil
}

// HTTPTransport communicates with an MCP server via StreamableHTTP.
type HTTPTransport struct {
	baseURL string
	id      int64
	client  *http.Client
}

// NewHTTPTransport creates a StreamableHTTP-based MCP transport.
func NewHTTPTransport(baseURL string) *HTTPTransport {
	return &HTTPTransport{
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  &http.Client{},
	}
}

// Send posts a JSON-RPC request to the HTTP server.
func (t *HTTPTransport) Send(req *Request) (*Response, error) {
	t.id++
	req.ID = t.id
	req.JSONRPC = JSONRPCVersion

	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", t.baseURL, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	resp, err := t.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	var rpcResp Response
	if err := json.NewDecoder(resp.Body).Decode(&rpcResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if rpcResp.Error != nil {
		return &rpcResp, fmt.Errorf("rpc error %d: %s", rpcResp.Error.Code, rpcResp.Error.Message)
	}

	return &rpcResp, nil
}

func (t *HTTPTransport) Close() error {
	t.client.CloseIdleConnections()
	return nil
}
