package params

import (
	goflag "flag"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGoFlags(t *testing.T) {
	gofs := goflag.NewFlagSet("TestGoFlags", goflag.ContinueOnError)
	stringFlag := gofs.String("string-flag", "stringFlag", "string")
	boolFlag := gofs.Bool("bool-flag", false, "bool")
	fs := FlagSetFromGoFlagSet(gofs)
	err := fs.Parse([]string{"--string-flag=bob", "--bool-flag"})
	require.NoError(t, err)
	assert.Equal(t, "bob", *stringFlag)
	assert.True(t, *boolFlag)
}

func TestGoFlagsEnviron(t *testing.T) {
	gofs := goflag.NewFlagSet("TestGoFlagsEnviron", goflag.ContinueOnError)
	stringFlag := gofs.String("string-flag", "stringFlag", "string")
	boolFlag := gofs.Bool("bool-flag", false, "bool")
	fs := FlagSetFromGoFlagSet(gofs)
	err := fs.ParseEnv([]string{"STRING_FLAG=bob", "BOOL_FLAG=true"})
	require.NoError(t, err)
	assert.Equal(t, "bob", *stringFlag)
	assert.True(t, *boolFlag)
}
