package client

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	cmap "github.com/orcaman/concurrent-map/v2"

	"github.com/ThinkInAIXYZ/go-mcp/pkg"
	"github.com/ThinkInAIXYZ/go-mcp/protocol"
	"github.com/ThinkInAIXYZ/go-mcp/transport"
)

type Option func(*Client)

func WithToolsListChangedNotifyHandler(handler func(ctx context.Context, request *protocol.ToolListChangedNotification) error) Option {
	return func(s *Client) {
		s.notifyHandlerWithToolsListChanged = handler
	}
}

func WithPromptListChangedNotifyHandler(handler func(ctx context.Context, request *protocol.PromptListChangedNotification) error) Option {
	return func(s *Client) {
		s.notifyHandlerWithPromptListChanged = handler
	}
}

func WithResourceListChangedNotifyHandler(handler func(ctx context.Context, request *protocol.ResourceListChangedNotification) error) Option {
	return func(s *Client) {
		s.notifyHandlerWithResourceListChanged = handler
	}
}

func WithResourcesUpdatedNotifyHandler(handler func(ctx context.Context, request *protocol.ResourceUpdatedNotification) error) Option {
	return func(s *Client) {
		s.notifyHandlerWithResourcesUpdated = handler
	}
}

func WithClientInfo(info protocol.Implementation) Option {
	return func(s *Client) {
		s.clientInfo = &info
	}
}

func WithInitTimeout(timeout time.Duration) Option {
	return func(s *Client) {
		s.initTimeout = timeout
	}
}

func WithLogger(logger pkg.Logger) Option {
	return func(s *Client) {
		s.logger = logger
	}
}

type Client struct {
	transport transport.ClientTransport

	reqID2respChan cmap.ConcurrentMap[string, chan *protocol.JSONRPCResponse]

	notifyHandlerWithToolsListChanged    func(ctx context.Context, request *protocol.ToolListChangedNotification) error
	notifyHandlerWithPromptListChanged   func(ctx context.Context, request *protocol.PromptListChangedNotification) error
	notifyHandlerWithResourceListChanged func(ctx context.Context, request *protocol.ResourceListChangedNotification) error
	notifyHandlerWithResourcesUpdated    func(ctx context.Context, request *protocol.ResourceUpdatedNotification) error

	requestID int64

	ready *pkg.AtomicBool

	clientInfo         *protocol.Implementation
	clientCapabilities *protocol.ClientCapabilities

	serverCapabilities *protocol.ServerCapabilities
	serverInfo         *protocol.Implementation
	serverInstructions string

	initTimeout time.Duration

	closed chan struct{}

	logger pkg.Logger
}

func NewClient(t transport.ClientTransport, opts ...Option) (*Client, error) {
	client := &Client{
		transport:          t,
		reqID2respChan:     cmap.New[chan *protocol.JSONRPCResponse](),
		ready:              pkg.NewAtomicBool(),
		clientInfo:         &protocol.Implementation{},
		clientCapabilities: &protocol.ClientCapabilities{},
		initTimeout:        time.Second * 30,
		closed:             make(chan struct{}),
		logger:             pkg.DefaultLogger,
	}
	t.SetReceiver(transport.ClientReceiverF(client.receive))

	for _, opt := range opts {
		opt(client)
	}

	if client.notifyHandlerWithToolsListChanged == nil {
		client.notifyHandlerWithToolsListChanged = func(_ context.Context, notify *protocol.ToolListChangedNotification) error {
			return defaultNotifyHandler(client.logger, notify)
		}
	}

	if client.notifyHandlerWithPromptListChanged == nil {
		client.notifyHandlerWithPromptListChanged = func(_ context.Context, notify *protocol.PromptListChangedNotification) error {
			return defaultNotifyHandler(client.logger, notify)
		}
	}

	if client.notifyHandlerWithResourceListChanged == nil {
		client.notifyHandlerWithResourceListChanged = func(_ context.Context, notify *protocol.ResourceListChangedNotification) error {
			return defaultNotifyHandler(client.logger, notify)
		}
	}

	if client.notifyHandlerWithResourcesUpdated == nil {
		client.notifyHandlerWithResourcesUpdated = func(_ context.Context, notify *protocol.ResourceUpdatedNotification) error {
			return defaultNotifyHandler(client.logger, notify)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), client.initTimeout)
	defer cancel()

	if err := client.transport.Start(); err != nil {
		return nil, fmt.Errorf("init mcp client transpor start fail: %w", err)
	}

	if _, err := client.initialization(ctx, protocol.NewInitializeRequest(*client.clientInfo, *client.clientCapabilities)); err != nil {
		return nil, err
	}

	go func() {
		defer pkg.Recover()

		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()

		select {
		case <-client.closed:
			return
		case <-ticker.C:
			if _, err := client.Ping(ctx, protocol.NewPingRequest()); err != nil {
				client.logger.Warnf("mcp client ping server fail: %v", err)
			}
		}
	}()

	return client, nil
}

func (client *Client) GetServerCapabilities() protocol.ServerCapabilities {
	return *client.serverCapabilities
}

func (client *Client) GetServerInfo() protocol.Implementation {
	return *client.serverInfo
}

func (client *Client) GetServerInstructions() string {
	return client.serverInstructions
}

func (client *Client) Close() error {
	close(client.closed)

	return client.transport.Close()
}

func defaultNotifyHandler(logger pkg.Logger, notify interface{}) error {
	b, err := json.Marshal(notify)
	if err != nil {
		return err
	}
	logger.Infof("receive notify: %s", b)
	return nil
}
