package models

type PortBinding struct {
	PublicPort  string
	PrivatePort string
	IP          string
	Protocol    string
}
