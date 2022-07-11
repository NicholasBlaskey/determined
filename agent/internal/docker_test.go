package internal

import (
	//"fmt"
	"testing"

	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/docker/distribution/reference"
	"github.com/docker/docker/api/types"

	"github.com/stretchr/testify/require"
)

func TestGetDockerAuths(t *testing.T) {
	dockerhubAuthConfig := types.AuthConfig{
		Username:      "username",
		Password:      "password",
		ServerAddress: "docker.io",
	}

	exampleDockerConfig := types.AuthConfig{
		Auth:          "token",
		ServerAddress: "https://example.com",
	}

	noServerAuthConfig := types.AuthConfig{
		Username: "username",
		Password: "password",
	}

	cases := []struct {
		image            string
		expconfReg       *types.AuthConfig
		credentialStores map[string]*credentialStore
		authConfigs      map[string]types.AuthConfig
		expected         types.AuthConfig
	}{
		// No authentication passed in.
		{"detai", nil, nil, nil, types.AuthConfig{}},
		// Correct server passed in for dockerhub.
		{"detai", &dockerhubAuthConfig, nil, nil, dockerhubAuthConfig},
		// Correct server passed in for example.com.
		{"example.com/detai", &exampleDockerConfig, nil, nil, exampleDockerConfig},
		// Different server passed than specified auth.
		{"example.com/detai", &dockerhubAuthConfig, nil, nil, types.AuthConfig{}},
		// No server (behaviour is deprecated)
		{"example.com/detai", &noServerAuthConfig, nil, nil, types.AuthConfig{}},
		{"example.com/detai", &noServerAuthConfig, nil, nil, types.AuthConfig{Username: "FALSE"}},

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

		/*
			// Mock an actor to just get an actor.Context with a nonnil sender.
			system := actor.NewSystem(fmt.Sprintf("%d", i))
			system.ActorOf(actor.Addr("test"), actor.ActorFunc(func(ctx *actor.Context) error {
				fmt.Println("CONTEXT", ctx.Message())

				actual, err := d.getDockerAuths(ctx, testCase.expconfReg, ref)
				require.NoError(t, err)
				require.Equal(t, testCase.expected, actual)

				panic("HERE")
				return nil
			}))
			//system.Ask(r, "").Get()
		*/
	}
}
