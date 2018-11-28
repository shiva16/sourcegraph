package conftypes

import "reflect"

// DeploymentConfiguration represents configuration about the deployment. These
// are settings that need to be propagated from the frontend to other services,
// so that the frontend can be the source of truth for all configuration.
type DeploymentConfiguration struct {
	// GitServers is the addresses of gitserver instances that should be talked
	// to.
	GitServers []string `json:"gitServers"`
}

// RawUnifiedConfiguration is the unparsed variant of conf.UnifiedConfiguration.
type RawUnifiedConfiguration struct {
	Site, Core string
	Deployment DeploymentConfiguration
}

func (r RawUnifiedConfiguration) Equal(other RawUnifiedConfiguration) bool {
	return r.Site == other.Site && r.Core == other.Core && reflect.DeepEqual(r.Deployment, other.Deployment)
}
