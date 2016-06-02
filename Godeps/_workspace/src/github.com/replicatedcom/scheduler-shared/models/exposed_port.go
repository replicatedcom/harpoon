package models

type ExposedPort struct {
	PublicPort  string
	PrivatePort string
	Interface   string
	Protocol    string
}
