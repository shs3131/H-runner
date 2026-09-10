package protocol

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Microsoft/go-winio"
)

// Handler processes an incoming launch request and returns response / steps.
type Handler interface {
	HandleLaunchRequest(req *LaunchRequest, conn *ClientConn) (*LaunchResponse, error)
	OnClientConnected()
	OnClientDisconnected()
}

// ClientConn wraps a single connected IPC client.
type ClientConn struct {
	conn    net.Conn
	writer  *bufio.Writer
	writeMu sync.Mutex
}

// SendMessage sends a JSON-encoded message followed by a newline.
func (c *ClientConn) SendMessage(v interface{}) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	if _, err := c.writer.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}
	return c.writer.Flush()
}

// Close terminates the client connection.
func (c *ClientConn) Close() error {
	return c.conn.Close()
}

// Server manages the Hrunner local IPC Named Pipe server and auto-shutdown lifecycle.
type Server struct {
	pipePath       string
	listener       net.Listener
	handler        Handler
	idleTimeout    time.Duration
	activeClients  int64
	activeOps      int64
	idleTimer      *time.Timer
	idleMu         sync.Mutex
	shutdownChan   chan struct{}
	ctx            context.Context
	cancel         context.CancelFunc
	wg             sync.WaitGroup
	serverMu       sync.Mutex
	isClosed       bool
}

// NewServer creates a new Hrunner IPC server.
// If pipePath is empty, DefaultPipeName is used.
// If idleTimeout is 0, a default of 5 seconds is used.
func NewServer(pipePath string, idleTimeout time.Duration, handler Handler) *Server {
	if pipePath == "" {
		pipePath = DefaultPipeName
	}
	if idleTimeout <= 0 {
		idleTimeout = 5 * time.Second
	}

	ctx, cancel := context.WithCancel(context.Background())
	s := &Server{
		pipePath:     pipePath,
		handler:      handler,
		idleTimeout:  idleTimeout,
		shutdownChan: make(chan struct{}),
		ctx:          ctx,
		cancel:       cancel,
	}

	return s
}

// Start begins listening on the named pipe and starts the idle manager.
func (s *Server) Start() error {
	cfg := &winio.PipeConfig{
		SecurityDescriptor: "", // Default DACL allows current user
		MessageMode:        false,
		InputBufferSize:    65536,
		OutputBufferSize:   65536,
	}

	l, err := winio.ListenPipe(s.pipePath, cfg)
	if err != nil {
		return fmt.Errorf("failed to listen on pipe %s: %w", s.pipePath, err)
	}

	s.listener = l
	s.resetIdleTimer() // Start initial idle countdown in case no client connects

	s.wg.Add(1)
	go s.acceptLoop()

	return nil
}

// AddOperation informs the server that an active operation (e.g. GUI open, install) is occurring.
// This prevents auto-shutdown while operations are ongoing.
func (s *Server) AddOperation() {
	atomic.AddInt64(&s.activeOps, 1)
	s.stopIdleTimer()
}

// DoneOperation decrements active operation count. If 0 and no clients, starts idle timer.
func (s *Server) DoneOperation() {
	ops := atomic.AddInt64(&s.activeOps, -1)
	if ops < 0 {
		atomic.StoreInt64(&s.activeOps, 0)
	}
	s.checkIdle()
}

func (s *Server) stopIdleTimer() {
	s.idleMu.Lock()
	defer s.idleMu.Unlock()
	if s.idleTimer != nil {
		s.idleTimer.Stop()
	}
}

func (s *Server) resetIdleTimer() {
	s.idleMu.Lock()
	defer s.idleMu.Unlock()

	if s.idleTimer != nil {
		s.idleTimer.Stop()
	}

	s.idleTimer = time.AfterFunc(s.idleTimeout, func() {
		clients := atomic.LoadInt64(&s.activeClients)
		ops := atomic.LoadInt64(&s.activeOps)
		if clients == 0 && ops == 0 {
			// Auto-shutdown
			s.Close()
		}
	})
}

func (s *Server) checkIdle() {
	clients := atomic.LoadInt64(&s.activeClients)
	ops := atomic.LoadInt64(&s.activeOps)
	if clients == 0 && ops == 0 {
		s.resetIdleTimer()
	} else {
		s.stopIdleTimer()
	}
}

func (s *Server) acceptLoop() {
	defer s.wg.Done()

	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.ctx.Done():
				return
			default:
				// Listener closed or failed
				return
			}
		}

		atomic.AddInt64(&s.activeClients, 1)
		s.stopIdleTimer()

		if s.handler != nil {
			s.handler.OnClientConnected()
		}

		s.wg.Add(1)
		go s.handleClient(conn)
	}
}

func (s *Server) handleClient(conn net.Conn) {
	defer func() {
		conn.Close()
		atomic.AddInt64(&s.activeClients, -1)
		if s.handler != nil {
			s.handler.OnClientDisconnected()
		}
		s.checkIdle()
		s.wg.Done()
	}()

	clientConn := &ClientConn{
		conn:   conn,
		writer: bufio.NewWriter(conn),
	}

	reader := bufio.NewReader(conn)
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if errors.Is(err, io.EOF) {
				return
			}
			return
		}

		if len(line) == 0 {
			continue
		}

		base, err := DecodeBaseMessage(line)
		if err != nil {
			_ = clientConn.SendMessage(&LaunchResponse{
				BaseMessage:  BaseMessage{ProtocolVersion: CurrentProtocolVersion, Type: MsgLaunchResponse},
				Status:       StatusError,
				ErrorMessage: fmt.Sprintf("invalid message format: %v", err),
			})
			continue
		}

		switch base.Type {
		case MsgPing:
			_ = clientConn.SendMessage(&BaseMessage{
				ProtocolVersion: CurrentProtocolVersion,
				Type:            MsgPong,
			})
		case MsgShutdownRequest:
			_ = clientConn.SendMessage(&BaseMessage{
				ProtocolVersion: CurrentProtocolVersion,
				Type:            MsgShutdownResponse,
			})
			go s.Close()
			return
		case MsgLaunchRequest:
			var req LaunchRequest
			if err := json.Unmarshal(line, &req); err != nil {
				_ = clientConn.SendMessage(&LaunchResponse{
					BaseMessage:  BaseMessage{ProtocolVersion: CurrentProtocolVersion, Type: MsgLaunchResponse},
					Status:       StatusError,
					ErrorMessage: fmt.Sprintf("invalid launch request: %v", err),
				})
				continue
			}

			if s.handler != nil {
				resp, err := s.handler.HandleLaunchRequest(&req, clientConn)
				if err != nil {
					_ = clientConn.SendMessage(&LaunchResponse{
						BaseMessage:  BaseMessage{ProtocolVersion: CurrentProtocolVersion, Type: MsgLaunchResponse},
						Status:       StatusError,
						ErrorMessage: err.Error(),
					})
				} else if resp != nil {
					_ = clientConn.SendMessage(resp)
				}
			}
		}
	}
}

// WaitShutdown blocks until the server has been closed or context cancelled.
func (s *Server) WaitShutdown() {
	select {
	case <-s.shutdownChan:
	case <-s.ctx.Done():
	}
	s.wg.Wait()
}

// ShutdownChan returns the notification channel for server exit.
func (s *Server) ShutdownChan() <-chan struct{} {
	return s.shutdownChan
}

// Close immediately shuts down the listener and all active sessions.
func (s *Server) Close() error {
	s.serverMu.Lock()
	if s.isClosed {
		s.serverMu.Unlock()
		return nil
	}
	s.isClosed = true
	s.serverMu.Unlock()

	s.cancel()
	s.stopIdleTimer()

	var err error
	if s.listener != nil {
		err = s.listener.Close()
	}

	close(s.shutdownChan)
	return err
}
