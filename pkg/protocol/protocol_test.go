package protocol

import (
	"fmt"
	"math/rand"
	"testing"
	"time"
)

type mockHandler struct {
	connected    int
	disconnected int
	reqReceived  *LaunchRequest
}

func (m *mockHandler) HandleLaunchRequest(req *LaunchRequest, conn *ClientConn) (*LaunchResponse, error) {
	m.reqReceived = req
	return &LaunchResponse{
		BaseMessage:     BaseMessage{ProtocolVersion: CurrentProtocolVersion, Type: MsgLaunchResponse},
		Status:          StatusReady,
		PythonAvailable: true,
	}, nil
}

func (m *mockHandler) OnClientConnected() {
	m.connected++
}

func (m *mockHandler) OnClientDisconnected() {
	m.disconnected++
}

func randomPipe() string {
	return fmt.Sprintf(`\\.\pipe\hrunner_test_%d_%d`, time.Now().UnixNano(), rand.Intn(10000))
}

func TestNamedPipeRoundTrip(t *testing.T) {
	pipePath := randomPipe()
	handler := &mockHandler{}

	server := NewServer(pipePath, 5*time.Second, handler)
	if err := server.Start(); err != nil {
		t.Fatalf("failed to start server: %v", err)
	}
	defer server.Close()

	if !IsServerRunning(pipePath, 2*time.Second) {
		t.Fatal("expected server to be running")
	}

	client, err := Dial(pipePath, 2*time.Second)
	if err != nil {
		t.Fatalf("failed to dial server: %v", err)
	}
	defer client.Close()

	// Test Ping/Pong
	if err := client.Ping(2 * time.Second); err != nil {
		t.Fatalf("ping failed: %v", err)
	}

	// Test LaunchRequest
	req := &LaunchRequest{
		ApplicationID: "com.test.app",
		Name:          "Test App",
		Version:       "1.0.0",
		Python:        "3.13.7",
		Dependencies:  map[string]string{"requests": "2.32.5"},
		Entrypoint:    "main.py",
	}

	resp, err := client.SendLaunchRequest(req)
	if err != nil {
		t.Fatalf("SendLaunchRequest failed: %v", err)
	}

	if resp.Status != StatusReady {
		t.Errorf("expected StatusReady, got %s", resp.Status)
	}
	if !resp.PythonAvailable {
		t.Errorf("expected PythonAvailable true")
	}
	if handler.reqReceived == nil || handler.reqReceived.ApplicationID != "com.test.app" {
		t.Errorf("expected server handler to receive com.test.app")
	}
}

func TestAutoShutdownOnIdle(t *testing.T) {
	pipePath := randomPipe()
	handler := &mockHandler{}

	// Set very short idle timeout: 300ms
	server := NewServer(pipePath, 300*time.Millisecond, handler)
	if err := server.Start(); err != nil {
		t.Fatalf("failed to start server: %v", err)
	}

	client, err := Dial(pipePath, 2*time.Second)
	if err != nil {
		t.Fatalf("failed to connect client: %v", err)
	}

	// Ping while connected
	if err := client.Ping(1 * time.Second); err != nil {
		t.Fatalf("ping failed: %v", err)
	}

	// Disconnect client
	client.Close()

	// Server should shut down within ~500ms
	select {
	case <-server.ShutdownChan():
		// Successfully auto-shutdown
	case <-time.After(2 * time.Second):
		t.Fatal("server failed to auto-shutdown after idle timeout")
	}
}
