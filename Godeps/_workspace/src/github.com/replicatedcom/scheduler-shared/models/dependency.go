package models

import (
	"fmt"
)

type Dependency struct {
	EventName string
	Image     string
}

func GetCommitDependency(id string) Dependency {
	return Dependency{
		EventName: fmt.Sprintf("commit-%s", id),
	}
}
