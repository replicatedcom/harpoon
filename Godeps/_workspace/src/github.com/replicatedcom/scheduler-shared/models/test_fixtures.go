package models

import (
	"net"
	"time"

	"github.com/replicatedcom/preflight/sig"
	"github.com/replicatedcom/preflight/utils"

	"github.com/blang/semver"
)

func containerStateFixture() ContainerState {
	node := nodeFixture()
	return ContainerState{
		ID:    "52641f643020862b8fa9c4068b6c9642683fb380eff0a17503819ba9d29b7a57",
		Image: "1a908c5b60b5b738902d29f03f55b25e9f1a128e7ded4374608d22777fc38bff",
		Labels: map[string]string{
			"ReplicatedClusterID":  "012cf14f655320bcf426ea2e7114eced",
			"ReplicatedVolumesDir": "432002cc88ba62a8027d755b4a2478fa",
		},
		State: stateFixture(),
		PortBindings: []PortBinding{
			PortBinding{
				PublicPort:  "33062",
				PrivatePort: "443",
				IP:          "0.0.0.0",
				Protocol:    "tcp",
			},
			PortBinding{
				PublicPort:  "80",
				PrivatePort: "80",
				IP:          "0.0.0.0",
				Protocol:    "tcp",
			},
		},
		Node: &node,
		Time: int64(1455581734),
	}
}

func clusterNodeStateFixture() ClusterNodeState {
	return ClusterNodeState{
		Node:          nodeFixture(),
		Version:       "1.0.0",
		RemoteAddress: "172.17.0.1:60459",
		IsConnected:   true,
		IsInitialized: true,
	}
}

func eventFixture() Event {
	return Event{
		Type:  EventTypeContainerPause,
		Name:  "Container nginx paused:52641f643020862b8fa9c4068b6c9642683fb380eff0a17503819ba9d29b7a57",
		Image: "192.168.134.186:9874/nginx:latest",
		Data: map[string]interface{}{
			"EventName": "Container nginx paused",
		},
	}
}

func stateFixture() State {
	return State{
		Running:    true,
		Paused:     false,
		Restarting: false,
		OOMKilled:  false,
		Pid:        30157,
		ExitCode:   0,
		Error:      "",
		StartedAt:  time.Date(2016, time.February, 16, 0, 15, 35, 113132462, time.UTC),
		FinishedAt: time.Time{}.UTC(),
	}
}

func nodeFixture() Node {
	return Node{
		ID:             "06c2083a50abff303657296c1ca8bb3e",
		Tags:           []string{"db"},
		PrivateAddress: net.ParseIP("192.168.134.186"),
		PublicAddress:  net.ParseIP("192.168.100.102"),
		DockerAddress:  net.ParseIP("172.17.0.1"),
		InterfaceAddresses: map[string]net.IP{
			"eth0": net.ParseIP("172.17.0.4"),
		},
		ContainerID: "d66134f16fd148058ca655bbc5465778",
		SystemInfo:  systemInfoFixture(),
	}
}

func systemInfoFixture() sig.SystemInfo {
	return sig.SystemInfo{
		Platform:            "linux",
		PlaformVersion:      "15.10 (Wily Werewolf)",
		PlaformVersionID:    "15.10",
		LinuxDistribution:   "Ubuntu",
		LinuxDistributionID: "ubuntu",
		LinuxKernelVersion: utils.Version{
			Version:    semver.MustParse("4.2.0"),
			VersionStr: "4.2.0-34-generic",
		},
		MemoryBytes: 4145475584,
		CPUCores:    2,
		DockerVersion: utils.Version{
			Version:    semver.MustParse("1.9.1"),
			VersionStr: "1.9.1",
		},
		DockerDriverRootDir: "/var/lib/docker/aufs",
	}
}
