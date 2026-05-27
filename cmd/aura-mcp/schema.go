package main

import "encoding/json"

// JSONRPCRequest represents a standard incoming JSON-RPC 2.0 request
type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
	ID      interface{}     `json:"id,omitempty"`
}

// JSONRPCResponse represents a standard success response
type JSONRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	Result  interface{} `json:"result,omitempty"`
	ID      interface{} `json:"id"`
}

// JSONRPCErrorResponse represents an error response
type JSONRPCErrorResponse struct {
	JSONRPC string        `json:"jsonrpc"`
	Error   JSONRPCError  `json:"error"`
	ID      interface{}   `json:"id"`
}

type JSONRPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Tool represents the definition of an MCP tool
type Tool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema InputSchema `json:"inputSchema"`
}

type InputSchema struct {
	Type       string                `json:"type"`
	Properties map[string]Property   `json:"properties"`
	Required   []string              `json:"required,omitempty"`
}

type Property struct {
	Type        string   `json:"type"`
	Description string   `json:"description"`
	Enum        []string `json:"enum,omitempty"`
}

// CallToolParams represents parameters for tools/call method
type CallToolParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments,omitempty"`
}

// ToolContent represents a block of content returned by an MCP tool call
type ToolContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type ToolCallResult struct {
	Content []ToolContent `json:"content"`
	IsError bool          `json:"isError,omitempty"`
}
