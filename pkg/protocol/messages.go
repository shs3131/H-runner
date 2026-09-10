package protocol

import (
	"encoding/json"
	"fmt"
)

// CurrentProtocolVersion defines the IPC protocol version.
const CurrentProtocolVersion = 1

// DefaultPipeName is the standard Windows named pipe used by Hrunner.
const DefaultPipeName = `\\.\pipe\hrunner`

// MessageType identifies the message payload.
type MessageType string

const (
	MsgLaunchRequest    MessageType = "launch_request"
	MsgLaunchResponse   MessageType = "launch_response"
	MsgInstallProgress  MessageType = "install_progress"
	MsgLaunchReady      MessageType = "launch_ready"
	MsgPing             MessageType = "ping"
	MsgPong             MessageType = "pong"
	MsgShutdownRequest  MessageType = "shutdown_request"
	MsgShutdownResponse MessageType = "shutdown_response"
)

// BaseMessage contains the envelope header for all IPC messages.
type BaseMessage struct {
	ProtocolVersion int         `json:"protocol_version"`
	Type            MessageType `json:"type"`
}

// MissingPackageInfo details a package that needs to be downloaded.
type MissingPackageInfo struct {
	Name          string `json:"name"`
	Version       string `json:"version"`
	DownloadBytes int64  `json:"download_bytes"`
}

// LaunchRequest is sent by the application launcher when starting up.
type LaunchRequest struct {
	BaseMessage
	ApplicationID         string            `json:"application_id"`
	Name                  string            `json:"name"`
	Version               string            `json:"version"`
	Python                string            `json:"python"`
	AllowCompatiblePython bool              `json:"allow_compatible_python,omitempty"`
	Dependencies          map[string]string `json:"dependencies"`
	Entrypoint            string            `json:"entrypoint"`
	ExecutablePath        string            `json:"executable_path"`
}

// LaunchStatus defines the state returned to the launcher.
type LaunchStatus string

const (
	StatusReady           LaunchStatus = "ready"
	StatusInstallRequired LaunchStatus = "install_required"
	StatusError           LaunchStatus = "error"
)

// LaunchResponse is returned by Hrunner after inspecting the environment.
type LaunchResponse struct {
	BaseMessage
	Status                 LaunchStatus         `json:"status"`
	PythonAvailable        bool                 `json:"python_available"`
	CurrentPythonVersion   string               `json:"current_python_version,omitempty"`
	MissingPython          bool                 `json:"missing_python"`
	RequestedPythonVersion string               `json:"requested_python_version,omitempty"`
	MissingPackages        []MissingPackageInfo `json:"missing_packages,omitempty"`
	TotalDownloadBytes     int64                `json:"total_download_bytes,omitempty"`
	ErrorMessage           string               `json:"error_message,omitempty"`
}

// ProgressStep describes the current stage of an installation operation.
type ProgressStep string

const (
	StepResolving           ProgressStep = "resolving"
	StepDownloadingPython   ProgressStep = "downloading_python"
	StepDownloadingPackages ProgressStep = "downloading_packages"
	StepExtracting          ProgressStep = "extracting"
	StepComplete            ProgressStep = "complete"
	StepFailed              ProgressStep = "failed"
)

// InstallProgress is streamed during dependency resolution and download.
type InstallProgress struct {
	BaseMessage
	Step           ProgressStep `json:"step"`
	ItemName       string       `json:"item_name,omitempty"`
	ItemIndex      int          `json:"item_index,omitempty"`
	TotalItems     int          `json:"total_items,omitempty"`
	BytesCompleted int64        `json:"bytes_completed,omitempty"`
	TotalBytes     int64        `json:"total_bytes,omitempty"`
	Percent        float64      `json:"percent"`
	Message        string       `json:"message"`
}

// LaunchReady is sent when all dependencies and Python runtime are prepared.
type LaunchReady struct {
	BaseMessage
	PythonExePath  string   `json:"python_exe_path"`
	PackagePaths   []string `json:"package_paths"` // directories to be included in sys.path
	ApplicationDir string   `json:"application_dir,omitempty"`
	Entrypoint     string   `json:"entrypoint"`
}

// EncodeMessage serializes any protocol struct into a newline-terminated JSON buffer.
func EncodeMessage(v interface{}) ([]byte, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("failed to encode message: %w", err)
	}
	return append(data, '\n'), nil
}

// DecodeBaseMessage inspects the type and version of an incoming JSON message.
func DecodeBaseMessage(data []byte) (*BaseMessage, error) {
	var base BaseMessage
	if err := json.Unmarshal(data, &base); err != nil {
		return nil, fmt.Errorf("failed to decode base message: %w", err)
	}
	if base.ProtocolVersion <= 0 {
		return nil, fmt.Errorf("missing or invalid protocol_version in message")
	}
	return &base, nil
}
