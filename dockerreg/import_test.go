package dockerreg

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestImportFromRemote(t *testing.T) {
	dockerRemote, err := ParseDockerURI("docker://redis:3.0.5")
	require.NoError(t, err)

	dockerRemote.Username = os.Getenv("DOCKERHUB_USERNAME")
	dockerRemote.Password = os.Getenv("DOCKERHUB_PASSWORD")

	err = ImportFromRemote(dockerRemote, "", false, "")
	require.NoError(t, err)
}
