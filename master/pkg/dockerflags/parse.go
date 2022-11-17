package dockerflags

import (
	"github.com/pkg/errors"
	"github.com/spf13/pflag"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	//"github.com/determined-ai/determined/master/pkg/cproto"
	//"github.com/determined-ai/determined/master/pkg/schemas"
)

// ParseDockerFlags runs same parsing as "docker run" to get Docker SDK structs.
func Parse(
	args []string,
) (*container.Config, *container.HostConfig, *network.NetworkingConfig, error) {
	if len(args) == 0 {
		return &container.Config{}, &container.HostConfig{}, &network.NetworkingConfig{}, nil
	}

	flagSet := pflag.NewFlagSet("parse", pflag.ContinueOnError)
	cOptions := addFlags(flagSet)
	if err := flagSet.Parse(args); err != nil {
		return nil, nil, nil, err
	}

	res, err := parse(flagSet, cOptions, "linux")
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "error parsing docker flags")
	}
	return res.Config, res.HostConfig, res.NetworkingConfig, nil
}

/*
// MergeIntoRunSpec parses dockerArgs and merges the result with the RunSpec.
func MergeIntoRunSpec(dockerArgs []string, runSpec cproto.RunSpec) (cproto.RunSpec, error) {
	if len(dockerArgs) == 0 {
		return runSpec, nil
	}

	conf, hostConf, networkConf, err := ParseDockerFlags(dockerArgs)
	if err != nil {
		return cproto.RunSpec{}, err
	}

	runSpec.ContainerConfig = schemas.Merge(runSpec.ContainerConfig, *conf).(container.Config)
	runSpec.HostConfig = schemas.Merge(runSpec.HostConfig, *hostConf).(container.HostConfig)
	runSpec.NetworkingConfig = schemas.Merge(
		runSpec.NetworkingConfig, *networkConf).(network.NetworkingConfig)

	return runSpec, nil
}
*/
