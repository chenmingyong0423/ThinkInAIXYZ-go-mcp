package client

import (
	"encoding/json"
	"github.com/ThinkInAIXYZ/go-mcp/protocol"
)

// BatchRequestItem represents an individual request item in a batch request
type BatchRequestItem struct {
	Method protocol.Method
	Params protocol.ClientRequest
}

// BatchRequestResult represents the result of a batch request
type BatchRequestResult struct {
	Result json.RawMessage
	Err    error
}
