// Package globals contains global variables that should be set by the frontend's main function on initialization.
package globals

import (
	"context"
	"net/url"

	"github.com/pkg/errors"
	"github.com/sourcegraph/sourcegraph/pkg/conf"
	"github.com/sourcegraph/sourcegraph/pkg/conf/confdb"
	"github.com/sourcegraph/sourcegraph/pkg/conf/conftypes"
)

// ExternalURL is the fully-resolved, externally accessible frontend URL.
var ExternalURL = &url.URL{Scheme: "http", Host: "example.com"}

// ConfigurationServerFrontendOnly provides the contents of the site configuration
// to other services and manages modifications to it.
//
// Any another service that attempts to use this variable will panic.
var ConfigurationServerFrontendOnly *conf.Server

func DoInit() {
	ConfigurationServerFrontendOnly = conf.InitConfigurationServerFrontendOnly(DBConfigurationSource)
}

// DBConfigurationSource is the configuration stored in the database.
var DBConfigurationSource = &configurationSource{}

type configurationSource struct{}

func (c configurationSource) Read(ctx context.Context) (conftypes.RawUnifiedConfiguration, error) {
	criticalFile, err := confdb.CriticalGetLatest(ctx)
	if err != nil {
		return conftypes.RawUnifiedConfiguration{}, errors.Wrap(err, "confdb.CriticalGetLatest")
	}
	siteFile, err := confdb.SiteGetLatest(ctx)
	if err != nil {
		return conftypes.RawUnifiedConfiguration{}, errors.Wrap(err, "confdb.SiteGetLatest")
	}
	// TODO(slimsag): UnifiedConfiguration
	deployment := conftypes.DeploymentConfiguration{}
	return conftypes.RawUnifiedConfiguration{
		Core:       criticalFile.Contents,
		Site:       siteFile.Contents,
		Deployment: deployment,
	}, nil
}

func (c configurationSource) Write(ctx context.Context, input conftypes.RawUnifiedConfiguration) error {
	// TODO(slimsag): UnifiedConfiguration
	//return ioutil.WriteFile(c.FilePath(), []byte(input), 0600)
	return nil
}
