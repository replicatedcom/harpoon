package models

type HostVolume struct {
	HostPath             string
	Owner                string // TODO: not yet supported
	Permission           string // TODO: not yet supported
	IsExcludedFromBackup bool   // TODO: not yet supported
	MinDiskSpaceBytes    uint64
}
