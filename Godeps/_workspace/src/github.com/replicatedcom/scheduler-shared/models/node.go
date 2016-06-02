package models

import (
	"net"

	"github.com/replicatedcom/preflight/sig"
)

type Node struct {
	ID                 string
	Tags               []string
	PrivateAddress     net.IP
	PublicAddress      net.IP
	DockerAddress      net.IP
	InterfaceAddresses map[string]net.IP
	AvailableMetrics   map[string]CGroupMetric
	ContainerID        string
	SystemInfo         sig.SystemInfo
}
