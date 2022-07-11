package internal

import (
	"fmt"
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

	dockerAuthSection := map[string]types.AuthConfig{
		"https://index.docker.io/v1/": types.AuthConfig{
			Auth:          "dockerhubtoken",
			ServerAddress: "docker.io",
		},
		"example.com": types.AuthConfig{
			Auth:          "exampletoken",
			ServerAddress: "example.com",
		},
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
		{"detai", &noServerAuthConfig, nil, nil, noServerAuthConfig},
		{"example.com/detai", &noServerAuthConfig, nil, nil, noServerAuthConfig},

		// Credential stores.
		// TODO

		// Docker auth config gets used.
		{"detai", nil, nil, dockerAuthSection, dockerAuthSection["https://index.docker.io/v1/"]},
		// Expconf takes precedence over docker config.
		{"detai", &dockerhubAuthConfig, nil, dockerAuthSection, dockerhubAuthConfig},
		// We fallback to auths if docker hub has wrong server.
		{"example.com/detai", &dockerhubAuthConfig, nil, dockerAuthSection,
			dockerAuthSection["example.com"]},
		// We don't return a result if we don't have that serveraddress.
		{"determined.ai/detai", nil, nil, dockerAuthSection, types.AuthConfig{}},
	}

	ctx := getMockDockerActorCtx()
	for _, testCase := range cases {
		d := dockerActor{
			credentialStores: testCase.credentialStores,
			authConfigs:      testCase.authConfigs,
		}
		d.credentialStores = testCase.credentialStores
		d.authConfigs = testCase.authConfigs

		// Parse image to correct format.
		ref, err := reference.ParseNormalizedNamed(testCase.image)
		require.NoError(t, err, "could not get image to correct format")
		ref = reference.TagNameOnly(ref)

		actual, err := d.getDockerAuths(ctx, testCase.expconfReg, ref)
		require.NoError(t, err)
		require.Equal(t, testCase.expected, actual)
	}
}

func getMockDockerActorCtx() *actor.Context {
	var ctx *actor.Context
	sys := actor.NewSystem("")
	child, _ := sys.ActorOf(actor.Addr("child"), actor.ActorFunc(func(context *actor.Context) error {
		ctx = context
		return nil
	}))
	parent, _ := sys.ActorOf(actor.Addr("parent"), actor.ActorFunc(func(context *actor.Context) error {
		context.Ask(child, "").Get()
		return nil
	}))
	sys.Ask(parent, "").Get()

	fmt.Printf("context? %+v", ctx)
	return ctx
}
