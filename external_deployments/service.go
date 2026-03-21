package external_deployments

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/kiali/kiali/config"
	"github.com/kiali/kiali/external_deployments/gh"
	"github.com/kiali/kiali/external_deployments/model"
	"github.com/kiali/kiali/log"
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
		if os.Getenv("TEST") == "true" {
			log.Info("using mock GitHub client")
			deploymentClient = gh.NewMockGithubClient()
		}
		return gh.NewGithubDeploymentService(deploymentClient, repo)
	}

	return nil, fmt.Errorf("external deployments provider %s not supported ", provider)
}
