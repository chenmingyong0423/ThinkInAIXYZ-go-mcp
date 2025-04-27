package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"sync/atomic"

	"github.com/ThinkInAIXYZ/go-mcp/pkg"
	"github.com/ThinkInAIXYZ/go-mcp/protocol"
)

func (client *Client) initialization(ctx context.Context, request *protocol.InitializeRequest) (*protocol.InitializeResult, error) {
	request.ProtocolVersion = protocol.Version

	response, err := client.callServer(ctx, protocol.Initialize, request)
	if err != nil {
		return nil, err
	}
	var result protocol.InitializeResult
	if err = pkg.JSONUnmarshal(response, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if _, ok := protocol.SupportedVersion[result.ProtocolVersion]; !ok {
		return nil, fmt.Errorf("protocol version not supported, supported lastest version is %v", protocol.Version)
	}

	if err = client.sendNotification4Initialized(ctx); err != nil {
		return nil, fmt.Errorf("failed to send InitializedNotification: %w", err)
	}

	client.clientInfo = &request.ClientInfo
	client.clientCapabilities = &request.Capabilities

	client.serverInfo = &result.ServerInfo
	client.serverCapabilities = &result.Capabilities
	client.serverInstructions = result.Instructions

	client.ready.Store(true)
	return &result, nil
}

func (client *Client) Ping(ctx context.Context, request *protocol.PingRequest) (*protocol.PingResult, error) {
	response, err := client.callServer(ctx, protocol.Ping, request)
	if err != nil {
		return nil, err
	}

	var result protocol.PingResult
	if err := pkg.JSONUnmarshal(response, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}
	return &result, nil
}

func (client *Client) ListPrompts(ctx context.Context) (*protocol.ListPromptsResult, error) {
	if client.serverCapabilities.Prompts == nil {
		return nil, pkg.ErrServerNotSupport
	}

	response, err := client.callServer(ctx, protocol.PromptsList, protocol.NewListPromptsRequest())
	if err != nil {
		return nil, err
	}

	var result protocol.ListPromptsResult
	if err := pkg.JSONUnmarshal(response, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}
	return &result, nil
}

func (client *Client) GetPrompt(ctx context.Context, request *protocol.GetPromptRequest) (*protocol.GetPromptResult, error) {
	if client.serverCapabilities.Prompts == nil {
		return nil, pkg.ErrServerNotSupport
	}

	response, err := client.callServer(ctx, protocol.PromptsGet, request)
	if err != nil {
		return nil, err
	}

	var result protocol.GetPromptResult
	if err := pkg.JSONUnmarshal(response, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &result, nil
}

func (client *Client) ListResources(ctx context.Context) (*protocol.ListResourcesResult, error) {
	if client.serverCapabilities.Resources == nil {
		return nil, pkg.ErrServerNotSupport
	}

	response, err := client.callServer(ctx, protocol.ResourcesList, protocol.NewListResourcesRequest())
	if err != nil {
		return nil, err
	}

	var result protocol.ListResourcesResult
	if err = pkg.JSONUnmarshal(response, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}
	return &result, err
}

func (client *Client) ListResourceTemplates(ctx context.Context) (*protocol.ListResourceTemplatesResult, error) {
	if client.serverCapabilities.Resources == nil {
		return nil, pkg.ErrServerNotSupport
	}

	response, err := client.callServer(ctx, protocol.ResourceListTemplates, protocol.NewListResourceTemplatesRequest())
	if err != nil {
		return nil, err
	}

	var result protocol.ListResourceTemplatesResult
	if err := pkg.JSONUnmarshal(response, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}
	return &result, nil
}

func (client *Client) ReadResource(ctx context.Context, request *protocol.ReadResourceRequest) (*protocol.ReadResourceResult, error) {
	if client.serverCapabilities.Resources == nil {
		return nil, pkg.ErrServerNotSupport
	}

	response, err := client.callServer(ctx, protocol.ResourcesRead, request)
	if err != nil {
		return nil, err
	}

	var result protocol.ReadResourceResult
	if err := pkg.JSONUnmarshal(response, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}
	return &result, nil
}

func (client *Client) SubscribeResourceChange(ctx context.Context, request *protocol.SubscribeRequest) (*protocol.SubscribeResult, error) {
	if client.serverCapabilities.Resources == nil || !client.serverCapabilities.Resources.Subscribe {
		return nil, pkg.ErrServerNotSupport
	}

	response, err := client.callServer(ctx, protocol.ResourcesSubscribe, request)
	if err != nil {
		return nil, err
	}

	var result protocol.SubscribeResult
	if len(response) > 0 {
		if err = pkg.JSONUnmarshal(response, &result); err != nil {
			return nil, fmt.Errorf("failed to unmarshal response: %w", err)
		}
	}
	return &result, nil
}

func (client *Client) UnSubscribeResourceChange(ctx context.Context, request *protocol.UnsubscribeRequest) (*protocol.UnsubscribeResult, error) {
	if client.serverCapabilities.Resources == nil || !client.serverCapabilities.Resources.Subscribe {
		return nil, pkg.ErrServerNotSupport
	}

	response, err := client.callServer(ctx, protocol.ResourcesUnsubscribe, request)
	if err != nil {
		return nil, err
	}

	var result protocol.UnsubscribeResult
	if len(response) > 0 {
		if err = pkg.JSONUnmarshal(response, &result); err != nil {
			return nil, fmt.Errorf("failed to unmarshal response: %w", err)
		}
	}
	return &result, nil
}

func (client *Client) ListTools(ctx context.Context) (*protocol.ListToolsResult, error) {
	if client.serverCapabilities.Tools == nil {
		return nil, pkg.ErrServerNotSupport
	}

	response, err := client.callServer(ctx, protocol.ToolsList, protocol.NewListToolsRequest())
	if err != nil {
		return nil, err
	}

	var result protocol.ListToolsResult
	if err := pkg.JSONUnmarshal(response, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}
	return &result, nil
}

func (client *Client) CallTool(ctx context.Context, request *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
	if client.serverCapabilities.Tools == nil {
		return nil, pkg.ErrServerNotSupport
	}

	response, err := client.callServer(ctx, protocol.ToolsCall, request)
	if err != nil {
		return nil, err
	}

	var result protocol.CallToolResult
	if err := pkg.JSONUnmarshal(response, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}
	return &result, nil
}

func (client *Client) sendNotification4Initialized(ctx context.Context) error {
	return client.sendMsgWithNotification(ctx, protocol.NotificationInitialized, protocol.NewInitializedNotification())
}

// Responsible for request and response assembly
func (client *Client) callServer(ctx context.Context, method protocol.Method, params protocol.ClientRequest) (json.RawMessage, error) {
	if !client.ready.Load() && (method != protocol.Initialize && method != protocol.Ping) {
		return nil, errors.New("callServer: client not ready")
	}

	requestID := strconv.FormatInt(atomic.AddInt64(&client.requestID, 1), 10)
	respChan := make(chan *protocol.JSONRPCResponse, 1)
	client.reqID2respChan.Set(requestID, respChan)
	defer client.reqID2respChan.Remove(requestID)

	if err := client.sendMsgWithRequest(ctx, requestID, method, params); err != nil {
		return nil, fmt.Errorf("callServer: %w", err)
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case response := <-respChan:
		if err := response.Error; err != nil {
			return nil, pkg.NewResponseError(err.Code, err.Message, err.Data)
		}
		return response.RawResult, nil
	}
}

// CallBatch sends a batch of requests to the server and returns the batch results
// This is the public API for batch requests
func (client *Client) CallBatch(ctx context.Context, items []BatchRequestItem) ([]BatchRequestResult, error) {
	if !client.ready.Load() {
		return nil, errors.New("CallBatch: client not ready")
	}

	// Check server support for each method
	for _, item := range items {
		switch item.Method {
		case protocol.PromptsList, protocol.PromptsGet:
			if client.serverCapabilities.Prompts == nil {
				return nil, pkg.ErrServerNotSupport
			}
		case protocol.ResourcesList, protocol.ResourcesRead, protocol.ResourceListTemplates:
			if client.serverCapabilities.Resources == nil {
				return nil, pkg.ErrServerNotSupport
			}
		case protocol.ResourcesSubscribe, protocol.ResourcesUnsubscribe:
			if client.serverCapabilities.Resources == nil || !client.serverCapabilities.Resources.Subscribe {
				return nil, pkg.ErrServerNotSupport
			}
		case protocol.ToolsList, protocol.ToolsCall:
			if client.serverCapabilities.Tools == nil {
				return nil, pkg.ErrServerNotSupport
			}
		}
	}

	return client.callServerBatch(ctx, items)
}

// callServerBatch processes batch requests and returns batch responses
func (client *Client) callServerBatch(ctx context.Context, batchItems []BatchRequestItem) ([]BatchRequestResult, error) {
	if !client.ready.Load() {
		return nil, errors.New("callServerBatch: client not ready")
	}

	if len(batchItems) == 0 {
		return nil, errors.New("callServerBatch: batch items can't be empty")
	}

	// Prepare request and response channels
	batchRequests := make(protocol.JSONRPCBatchRequests, 0, len(batchItems))
	requestResults := make([]BatchRequestResult, len(batchItems))
	respChans := make([]chan *protocol.JSONRPCResponse, len(batchItems))
	requestIDs := make([]string, len(batchItems))

	// Create requests and set up response channels
	for i, item := range batchItems {
		requestID := strconv.FormatInt(atomic.AddInt64(&client.requestID, 1), 10)
		requestIDs[i] = requestID
		respChan := make(chan *protocol.JSONRPCResponse, 1)
		client.reqID2respChan.Set(requestID, respChan)
		respChans[i] = respChan

		req := protocol.NewJSONRPCRequest(requestID, item.Method, item.Params)
		batchRequests = append(batchRequests, req)
	}

	// Add a cleanup function to ensure resources are released regardless of success
	defer func() {
		for i, reqID := range requestIDs {
			client.reqID2respChan.Remove(reqID)
			if respChans[i] != nil {
				close(respChans[i])
			}
		}
	}()

	// Send batch requests
	if err := client.sendMsgWithBatchRequests(ctx, batchRequests); err != nil {
		return nil, fmt.Errorf("callServerBatch: %w", err)
	}

	// Wait for all responses
	for i, respChan := range respChans {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case response := <-respChan:
			if err := response.Error; err != nil {
				requestResults[i].Err = pkg.NewResponseError(err.Code, err.Message, err.Data)
			} else {
				requestResults[i].Result = response.RawResult
			}
		}
	}

	return requestResults, nil
}
