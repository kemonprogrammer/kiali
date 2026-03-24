package external_deployments

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/kiali/kiali/config"
	"github.com/kiali/kiali/external_deployments/github"
	"github.com/kiali/kiali/external_deployments/model"
	"github.com/kiali/kiali/log"
)

type DeploymentClient interface {
	ListDeploymentsInRange(ctx context.Context, from, to time.Time) ([]*model.Deployment, error)
	SetRepo(ctx context.Context, repo string) error
	GetRepo() string
}

func NewDeploymentClient(conf *config.Config) (DeploymentClient, error) {
	if !conf.ExternalServices.ExternalDeployments.Enabled {
		return nil, fmt.Errorf("external deployments not enabled")
	}

	provider := conf.ExternalServices.ExternalDeployments.Provider
	if provider == "github" {
		owner := conf.ExternalServices.ExternalDeployments.Auth.Username.String()
		if len(owner) == 0 {
			return nil, fmt.Errorf("external_service.external_deployments.auth.username not set in config")
		}

		ghAPI, err := github.NewAPI(conf)
		if err != nil {
			return nil, err
		}
		if os.Getenv("TEST") == "true" {
			log.Info("using mock GitHub client")
			ghAPI = github.NewMockAPI()
		}
		return github.NewDeploymentClient(ghAPI)
	}

	return nil, fmt.Errorf("external deployments provider %s not supported ", provider)
}
