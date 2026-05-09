// Package awsutil provides shared helpers for AWS MCP service wrappers.
package awsutil

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// JSONResult marshals v as pretty JSON and wraps it in an MCP tool result.
func JSONResult(v any) (*mcp.CallToolResult, any, error) {
	body, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return ErrResult(err.Error())
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(body)}},
	}, nil, nil
}

// ErrResult returns a tool result flagged as error with the given message.
func ErrResult(msg string) (*mcp.CallToolResult, any, error) {
	r := &mcp.CallToolResult{}
	r.SetError(fmt.Errorf("%s", msg))
	return r, nil, nil
}

// AWSErr wraps an AWS SDK error into an error tool result with a stable prefix.
func AWSErr(op string, err error) (*mcp.CallToolResult, any, error) {
	return ErrResult(fmt.Sprintf("AWS %s error: %v", op, err))
}

// ParseTime parses an ISO 8601 / RFC3339 time string.
func ParseTime(s string) (time.Time, error) {
	return time.Parse(time.RFC3339, s)
}
