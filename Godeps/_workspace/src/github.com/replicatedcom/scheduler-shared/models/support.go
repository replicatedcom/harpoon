package models

import (
	"io"
)

type NodeSupportBundle struct {
	ID         string
	Filename   string
	DataReader io.ReadCloser
}

type ContainerSupportFile struct {
	Filename string
}

type ContainerSupportCommand struct {
	Filename string
	Command  []string
}

type ContainerSupportBundle struct {
	ID         string
	NodeID     string
	Filename   string
	DataReader io.ReadCloser
}
