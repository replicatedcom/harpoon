package dockerreg

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestImportFromRemote(t *testing.T) {
	dockerRemote, err := ParseDockerURI("docker://redis:3.0.5")
	require.NoError(t, err)

	err = ImportFromRemote(dockerRemote, "")
	require.NoError(t, err)
}
