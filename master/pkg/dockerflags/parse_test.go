package dockerflags

import (
	//"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	//"github.com/determined-ai/determined/master/pkg/cproto"
)

func TestParseDockerFlags(t *testing.T) {
	conf, hostConf, networkConf, err := Parse([]string{"--cpu-shares", "1025"})
	require.NoError(t, err)
	require.NotNil(t, conf)
	require.Equal(t, int64(1025), hostConf.CPUShares)
	require.NotNil(t, networkConf)

	conf, hostConf, networkConf, err = Parse([]string{
		"--mac-address", "00:00:5e:00:53:af",
	})
	require.NoError(t, err)
	require.Equal(t, "00:00:5e:00:53:af", conf.MacAddress)
	require.NotNil(t, hostConf)
	require.NotNil(t, networkConf)

	_, _, _, err = Parse([]string{"--this-isnt-a-docker-flag", "false"})
	require.Error(t, err)

	_, _, _, err = Parse([]string{})
	require.NoError(t, err)

	// Set a value to a default value and ensure we get the same as initializing the
	// config struct how we used to. This is important to keep behaviour the same
	// so the CLI parsing doesn't add a bunch of fields when we just set one flag.
	conf, hostConf, networkConf, err = Parse([]string{"--shm-size", "0"})
	require.NoError(t, err)
	require.Equal(t, &container.Config{}, conf)
	require.Equal(t, &container.HostConfig{}, hostConf)
	require.Equal(t, &network.NetworkingConfig{}, networkConf)
}

/*
func TestMergeIntoRunSpec(t *testing.T) {
	// Overwrite when runSpec has as zero value.
	runSpec, err := MergeIntoRunSpec([]string{"--cpu-shares", "42"}, cproto.RunSpec{})
	require.NoError(t, err)
	require.Equal(t, int64(42), runSpec.HostConfig.CPUShares)

	// Don't overwrite when runSpec has a value.
	runSpec = cproto.RunSpec{}
	runSpec.HostConfig.CPUShares = 63
	runSpec, err = MergeIntoRunSpec([]string{"--cpu-shares", "41"}, runSpec)
	require.NoError(t, err)
	require.Equal(t, int64(63), runSpec.HostConfig.CPUShares)
*/

// So we want a test that merging something returns nothing. Should cover defaulting behaviour we were worried about...

// Another thing worried about is like []string and maps getting merged wrong!
// Slices actrually work out. Only thing that doesn't is maps? do we actually set any maps
// is the big question. I'm assuming we do.

// TODO think about this. We can hard code just an int array can be returned I think
// It just is annoying since it is one uint[2] that is only used in windows...
// This might be irrelevant since map merges might be unacceptable.

// I mean the behaviour we really want is to like don't set values.
// I mean we want to get this runSpec and set all non zero values?

// Okay so first we want the dockerFlags spec.
// Then we want to write all non zero values in.

/*
	// TODO docker args
	var err error
	spec.RunSpec, err = dockerflags.MergeIntoRunSpec([]string{}, spec.RunSpec)
	if err != nil {
		return cproto.Spec{}, errors.Wrap(err, "error adding docker flags to Docker config")
	}
*/
//}
