package protocol

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/Microsoft/go-winio"
)

// Client handles IPC communication from the application launcher to Hrunner.
type Client struct {
	conn     net.Conn
	reader   *bufio.Reader
	writer   *bufio.Writer
	writeMu  sync.Mutex
	pipePath string
}

// IsServerRunning quickly checks if the Hrunner named pipe is available.
func IsServerRunning(pipePath string, timeout time.Duration) bool {
	if pipePath == "" {
		pipePath = DefaultPipeName
	}
	if timeout <= 0 {
		timeout = 300 * time.Millisecond
	}

	conn, err := winio.DialPipe(pipePath, &timeout)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

// Dial connects to the Hrunner named pipe server.
func Dial(pipePath string, timeout time.Duration) (*Client, error) {
	if pipePath == "" {
		pipePath = DefaultPipeName
	}
	if timeout <= 0 {
		timeout = 5 * time.Second
	}

	conn, err := winio.DialPipe(pipePath, &timeout)
	if err != nil {
		return nil, fmt.Errorf("failed to dial Hrunner pipe %s: %w", pipePath, err)
	}

	return &Client{
		conn:     conn,
		reader:   bufio.NewReader(conn),
		writer:   bufio.NewWriter(conn),
		pipePath: pipePath,
	}, nil
}

// Send sends any JSON serializable message followed by a newline.
func (c *Client) Send(v interface{}) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	if _, err := c.writer.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("failed to write message to pipe: %w", err)
	}
	return c.writer.Flush()
}

// ReadLine reads a single newline-delimited JSON line.
func (c *Client) ReadLine() ([]byte, error) {
	line, err := c.reader.ReadBytes('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to read from pipe: %w", err)
	}
	return line, nil
}

// SendLaunchRequest sends a LaunchRequest and awaits the initial LaunchResponse.
func (c *Client) SendLaunchRequest(req *LaunchRequest) (*LaunchResponse, error) {
	req.ProtocolVersion = CurrentProtocolVersion
	req.Type = MsgLaunchRequest

	if err := c.Send(req); err != nil {
		return nil, err
	}

	for {
		line, err := c.ReadLine()
		if err != nil {
			return nil, err
		}

		base, err := DecodeBaseMessage(line)
		if err != nil {
			return nil, err
		}

		if base.Type == MsgLaunchResponse {
			var resp LaunchResponse
			if err := json.Unmarshal(line, &resp); err != nil {
				return nil, fmt.Errorf("failed to unmarshal launch response: %w", err)
			}
			return &resp, nil
		}
	}
}

// Ping sends a ping and waits for pong.
func (c *Client) Ping(timeout time.Duration) error {
	if err := c.Send(&BaseMessage{ProtocolVersion: CurrentProtocolVersion, Type: MsgPing}); err != nil {
		return err
	}

	line, err := c.ReadLine()
	if err != nil {
		return err
	}

	base, err := DecodeBaseMessage(line)
	if err != nil {
		return err
	}

	if base.Type != MsgPong {
		return fmt.Errorf("expected pong, got: %s", base.Type)
	}

	return nil
}

// RequestShutdown requests Hrunner to terminate.
func (c *Client) RequestShutdown() error {
	if err := c.Send(&BaseMessage{ProtocolVersion: CurrentProtocolVersion, Type: MsgShutdownRequest}); err != nil {
		return err
	}

	line, err := c.ReadLine()
	if err != nil {
		return err
	}

	base, err := DecodeBaseMessage(line)
	if err != nil {
		return err
	}

	if base.Type != MsgShutdownResponse {
		return errors.New("unexpected response to shutdown request")
	}

	return nil
}

// Close closes the pipe connection.
func (c *Client) Close() error {
	return c.conn.Close()
}
