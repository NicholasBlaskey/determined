package internal

import (
	"testing"

	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/docker/distribution/reference"
	"github.com/docker/docker/api/types"

	"github.com/stretchr/testify/require"
)

func TestGetDockerAuths(t *testing.T) {
	cases := []struct {
		image            string
		expconfReg       *types.AuthConfig
		credentialStores map[string]*credentialStore
		authConfigs      map[string]types.AuthConfig
		expected         types.AuthConfig
	}{
		// No authentication passed in.
		{"detai", nil, nil, nil, types.AuthConfig{}},
		//
	}

	for _, testCase := range cases {
		d := dockerActor{
			credentialStores: testCase.credentialStores,
			authConfigs:      testCase.authConfigs,
		}
		ctx := &actor.Context{}

		// Parse image to correct format.
		ref, err := reference.ParseNormalizedNamed(testCase.image)
		require.NoError(t, err, "could not get image to correct format")
		ref = reference.TagNameOnly(ref)

		actual, err := d.getDockerAuths(ctx, testCase.expconfReg, ref)
		require.NoError(t, err)
		require.Equal(t, testCase.expected, actual)
	}
}
