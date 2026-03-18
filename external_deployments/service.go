package external_deployments

import (
	"context"
	"fmt"
	"time"

	"github.com/kiali/kiali/config"
	"github.com/kiali/kiali/external_deployments/gh"
	"github.com/kiali/kiali/external_deployments/model"
)

type DeploymentService interface {
	ListDeploymentsInRange(ctx context.Context, from, to time.Time) ([]*model.Deployment, error)
	ValidateRepo(ctx context.Context) error
}

func NewDeploymentService(conf *config.Config, repo string) (DeploymentService, error) {
	if !conf.ExternalServices.ExternalDeployments.Enabled {
		return nil, fmt.Errorf("external deployments not enabled")
	}

	provider := conf.ExternalServices.ExternalDeployments.Provider
	if provider == "github" {
		deploymentClient, err := gh.MakeGithubClientInterface(conf)
		if err != nil {
			return nil, err
		}
		return gh.NewGithubDeploymentService(deploymentClient, repo)
	}

	return nil, fmt.Errorf("external deployments provider %s not supported ", provider)
}
