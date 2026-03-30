package bridge

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"

	"github.com/grokify/mogo/log/slogutil"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Server is an MCP server that exposes bridge tools to agents.
type Server struct {
	bridge   *Bridge
	server   *mcp.Server
	listener net.Listener
	addr     string
	done     chan struct{}
}

// NewServer creates a new MCP server for bridge tools.
func NewServer(bridge *Bridge) *Server {
	s := &Server{
		bridge: bridge,
		done:   make(chan struct{}),
	}

	// Create MCP server
	s.server = mcp.NewServer(&mcp.Implementation{
		Name:    "agentpair-bridge",
		Version: "1.0.0",
	}, nil)

	// Register tools
	s.registerTools()

	return s
}

// Tool input types
type sendToAgentInput struct {
	To          string `json:"to" jsonschema:"Target agent name (claude or codex)"`
	MessageType string `json:"message_type" jsonschema:"Type of message: task, result, review, signal, or chat"`
	Content     string `json:"content" jsonschema:"Message content"`
	Signal      string `json:"signal,omitempty" jsonschema:"Signal value for signal type: DONE, PASS, or FAIL"`
}

type receiveMessagesInput struct {
	Agent   string `json:"agent" jsonschema:"Agent name to receive messages for (claude or codex)"`
	SinceID string `json:"since_id,omitempty" jsonschema:"Only return messages after this ID"`
}

type bridgeStatusInput struct{}

func (s *Server) registerTools() {
	// send_to_agent tool
	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "send_to_agent",
		Description: "Send a message to another agent through the bridge",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input sendToAgentInput) (*mcp.CallToolResult, any, error) {
		return s.handleSendToAgent(ctx, input)
	})

	// receive_messages tool
	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "receive_messages",
		Description: "Receive pending messages from the bridge",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input receiveMessagesInput) (*mcp.CallToolResult, any, error) {
		return s.handleReceiveMessages(ctx, input)
	})

	// bridge_status tool
	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "bridge_status",
		Description: "Get the current status of the bridge",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input bridgeStatusInput) (*mcp.CallToolResult, any, error) {
		return s.handleBridgeStatus(ctx)
	})
}

// textResult creates a CallToolResult with text content.
func textResult(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: text},
		},
	}
}

// errorResult creates a CallToolResult indicating an error.
func errorResult(text string) *mcp.CallToolResult {
	result := &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: text},
		},
		IsError: true,
	}
	return result
}

func (s *Server) handleSendToAgent(ctx context.Context, input sendToAgentInput) (*mcp.CallToolResult, any, error) {
	// Determine sender from context or default to "unknown"
	from := "unknown"

	var msg *Message
	switch MessageType(input.MessageType) {
	case TypeSignal:
		msg = NewSignalMessage(from, Signal(input.Signal), input.Content)
		msg.To = input.To
	default:
		msg = NewMessage(MessageType(input.MessageType), from, input.To, input.Content)
	}

	sent, err := s.bridge.Send(ctx, msg)
	if err != nil {
		return errorResult(fmt.Sprintf("send failed: %v", err)), nil, nil
	}

	result := "sent"
	if !sent {
		result = "duplicate (already sent)"
	}

	return textResult(fmt.Sprintf("Message %s: id=%s", result, msg.ID)), nil, nil
}

func (s *Server) handleReceiveMessages(ctx context.Context, input receiveMessagesInput) (*mcp.CallToolResult, any, error) {
	msgs, err := s.bridge.DrainNew(ctx, input.Agent, input.SinceID)
	if err != nil {
		return errorResult(fmt.Sprintf("receive failed: %v", err)), nil, nil
	}

	if len(msgs) == 0 {
		return textResult("No new messages"), nil, nil
	}

	// Format messages for display
	var text string
	for i, msg := range msgs {
		text += fmt.Sprintf("[%d] from=%s type=%s", i+1, msg.From, msg.Type)
		if msg.Signal != "" {
			text += fmt.Sprintf(" signal=%s", msg.Signal)
		}
		text += fmt.Sprintf("\n%s\n\n", msg.Content)
	}

	return textResult(text), nil, nil
}

func (s *Server) handleBridgeStatus(_ context.Context) (*mcp.CallToolResult, any, error) {
	status := s.bridge.Status()

	text := fmt.Sprintf("Bridge Status:\n"+
		"  Total Messages: %d\n"+
		"  Done Signal: %v\n"+
		"  Pass Count: %d\n"+
		"  Fail Count: %d\n"+
		"  Messages by Agent:\n",
		status.TotalMessages,
		status.HasDoneSignal,
		status.PassCount,
		status.FailCount)

	for agent, count := range status.ByAgent {
		text += fmt.Sprintf("    %s: %d\n", agent, count)
	}

	text += "  Messages by Type:\n"
	for msgType, count := range status.ByType {
		text += fmt.Sprintf("    %s: %d\n", msgType, count)
	}

	return textResult(text), nil, nil
}

// ListenAndServe starts the server on the given address.
// Use "stdio" for stdin/stdout transport, or a TCP address like ":0" for socket.
func (s *Server) ListenAndServe(ctx context.Context, addr string) error {
	if addr == "stdio" {
		return s.server.Run(ctx, &mcp.StdioTransport{})
	}
	return s.serveTCP(ctx, addr)
}

func (s *Server) serveTCP(ctx context.Context, addr string) error {
	var err error
	s.listener, err = net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}
	s.addr = s.listener.Addr().String()

	go func() {
		select {
		case <-ctx.Done():
			s.listener.Close()
		case <-s.done:
			s.listener.Close()
		}
	}()

	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.done:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			default:
				continue
			}
		}

		go s.handleConn(ctx, conn)
	}
}

func (s *Server) handleConn(ctx context.Context, conn net.Conn) {
	defer conn.Close()

	// net.Conn implements both io.ReadCloser and io.WriteCloser
	transport := &mcp.IOTransport{
		Reader: conn,
		Writer: conn,
	}

	// Run server session for this connection
	if _, err := s.server.Connect(ctx, transport, nil); err != nil {
		logger := slogutil.LoggerFromContext(ctx, slog.Default())
		logger.Debug("MCP connection ended", "error", err)
	}
}

// Addr returns the server's listen address (useful when using ":0").
func (s *Server) Addr() string {
	return s.addr
}

// Close shuts down the server.
func (s *Server) Close() error {
	close(s.done)
	if s.listener != nil {
		return s.listener.Close()
	}
	return nil
}

// ServeStdio runs the server over stdin/stdout.
func (s *Server) ServeStdio(ctx context.Context) error {
	return s.server.Run(ctx, &mcp.StdioTransport{})
}

// nopCloser wraps an io.Reader or io.Writer to add a no-op Close method.
type nopReadCloser struct {
	io.Reader
}

func (nopReadCloser) Close() error { return nil }

type nopWriteCloser struct {
	io.Writer
}

func (nopWriteCloser) Close() error { return nil }

// ServeIO runs the server over custom reader/writer.
func (s *Server) ServeIO(ctx context.Context, r io.Reader, w io.Writer) error {
	transport := &mcp.IOTransport{
		Reader: nopReadCloser{r},
		Writer: nopWriteCloser{w},
	}
	return s.server.Run(ctx, transport)
}
