package models

type ComponentsPlaceResponse struct {
	Component string
	Nodes     []*Node
}

type ComponentPlaceResponse struct {
	Component string
	Node      Node
}

type QueueContainerStartResponse struct {
	Node          *Node
	Image         string
	ClusterID     string
	InstanceCount uint
}

type ContainerStartResponse struct {
	ID           string
	PortBindings []PortBinding
}

type ClusterStateResponse struct {
	IsConnected      bool
	IsInitialized    bool
	ClusterNodeState []*ClusterNodeState
}
