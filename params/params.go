package params

import (
	goflag "flag"

	"github.com/blang/semver"
)

const (
	Version = "0.0.1"

	DefaultDockerCertsDir = "/etc/docker/certs.d/"
	DefaultDockerEndpoint = "unix:///var/run/docker.sock"
	DefaultLogLevel       = "info"
)

var (
	params = &Params{}
)

type Params struct {
	DockerCertsDir string
	DockerEndpoint string
	LogLevel       string
	Version        semver.Version
}

func init() {
	params.Version = semver.MustParse(Version)

	goflag.StringVar(&params.DockerCertsDir, "docker-certificates-directory", DefaultDockerCertsDir, "docker certificate directory (relative to the host's filesystem)")
	goflag.StringVar(&params.DockerEndpoint, "docker-endpoint", DefaultDockerEndpoint, "docker endpoint")
	goflag.StringVar(&params.LogLevel, "log-level", DefaultLogLevel, "log level")
}

func Get() *Params {
	return params
}

func Parse(args []string) error {
	FlagSetFromGoFlagSet(goflag.CommandLine).Parse(args)
	return nil
}
