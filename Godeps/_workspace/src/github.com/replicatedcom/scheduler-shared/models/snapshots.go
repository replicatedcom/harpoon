package models

import (
	"io"
)

type SnapshotVolume struct {
	ID               string
	NodeID           string
	ContainerID      string
	HostPath         string
	UncompressedSize uint64
	CompressedSize   uint64
	File             io.ReadCloser
	Filename         string
}
