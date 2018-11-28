package conf

import "github.com/sourcegraph/sourcegraph/pkg/conf/conftypes"

// TODO(slimsag): UnifiedConfiguration
var defaultDevAndTestingConfiguration = conftypes.RawUnifiedConfiguration{
	Core: `{}`,
	Site: `{}`,
}

// TODO(slimsag): UnifiedConfiguration
var defaultDockerContainerConfiguration = conftypes.RawUnifiedConfiguration{
	Core: `{}`,
	Site: `{}`,
}

// TODO(slimsag): UnifiedConfiguration
var defaultClusterConfiguration = conftypes.RawUnifiedConfiguration{
	Core: `{}`,
	Site: `{}`,
}
