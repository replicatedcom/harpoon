package models

type VolumeBinding struct {
	HostPath             string
	ContainerPath        string
	Owner                string // TODO: deprecate
	Permission           string // TODO: deprecate
	IsExcludedFromBackup bool   // TODO: deprecate
}
