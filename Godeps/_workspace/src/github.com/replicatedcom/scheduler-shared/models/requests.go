package models

import (
	"io"
	"time"
)

type ComponentsPlaceRequest struct {
	// TODO: I kinda feel like this should be ContainerCreateRequest in the future
	MinAPIVersion string
	Component     string
	Tags          []string
	Conflicts     []string
	MinCount      int
	MaxCount      int
	InstanceCount uint
	CanReallocate bool
	HostVolumes   []*HostVolume
}

type ComponentPlaceRequest struct {
	MinAPIVersion string
	Component     string
	Tags          []string
	Conflicts     []string
	InstanceCount uint
	NodeID        string
	HostVolumes   []*HostVolume
}

type QueueContainerStartRequest struct {
	Component    string
	Image        string
	MinCount     uint
	InitialCount uint
	Depends      []Dependency
}

type ContainerStartRequest struct {
	MinCount            uint
	InitialCount        uint
	BypassLocalRegistry bool
	Events              []Event
	Depends             []Dependency

	// Config
	Image    string
	Cmd      []string
	Env      []*Env
	Labels   map[string]string
	Tty      bool
	Hostname string

	// HostConfig
	ExposedPorts   []*ExposedPort
	VolumeBindings []*VolumeBinding
	ConfigFiles    []*ConfigFile
	ExtraHosts     []*ExtraHost
	Privileged     bool
	RestartPolicy  *RestartPolicy
	NetworkMode    string
	CapAdd         []string
	SecurityOpt    []string
	CPUShares      int64
	Memory         int64
	MemorySwap     int64
}

type ContainerStopRequest struct {
	Component string
	ID        string
	Image     string
	Timeout   uint
	Events    []Event
	Depends   []Dependency
}

type ContainerRemoveRequest struct {
	Component string
	ID        string
	Image     string
	Events    []Event
	Depends   []Dependency
}

type ContainerPauseRequest struct {
	Component string
	ID        string
	Image     string
	Events    []Event
	Depends   []Dependency
}

type ContainerUnpauseRequest struct {
	Component string
	ID        string
	Image     string
	Events    []Event
	Depends   []Dependency
}

type ContainerExecRequest struct {
	Component    string
	ID           string
	Image        string
	Command      []string
	Timeout      time.Duration
	InputStream  io.Reader
	OutputStream io.Writer
	ErrorStream  io.Writer
	ExitCode     int
}

type ContainerStateRequest struct {
	Component string
	ID        string
	Image     string
}

type ContainerMetricsRequest struct {
	Component string
	ID        string
	Image     string
	Metrics   []string
}

type CancelContainerMetricsRequest struct {
	Component string
	ID        string
	Image     string
}

type SnapshotContainerVolumesRequest struct {
	Component       string
	ID              string
	SnapshotID      string
	HostPath        string
	TmpPathOverride string
}

type SnapshotSendContainerVolumeRequest struct {
	NodeID          string
	Component       string
	ID              string
	HostPath        string
	Owner           string
	Permission      string
	FileData        io.ReadCloser
	TmpPathOverride string
}

type NodeSupportBundleRequest struct {
	ID string
}

type ContainerSupportBundleRequest struct {
	Component                string
	ID                       string
	ImageName                string
	ImageVersion             string
	ContainerSupportFiles    []*ContainerSupportFile
	ContainerSupportCommands []*ContainerSupportCommand
	MaskedContainerEnvVars   []string
}

type ConfigRequest struct {
	CA             []byte
	StatsdEndpoint string
}

type NodeConfigRequest struct {
	ID           string
	OverrideTags bool
	Tags         *[]string
}
