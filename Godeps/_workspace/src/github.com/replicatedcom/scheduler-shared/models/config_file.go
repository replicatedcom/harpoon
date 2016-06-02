package models

// Note: Not sure what "Owner" is, so I'm not going to remove it just yet
type ConfigFile struct {
	Filename  string
	Contents  string
	FileMode  string
	FileOwner string
	Owner     string
	Enabled   bool
}
