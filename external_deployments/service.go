package external_deployments

import (
	"context"
	"fmt"

	"github.com/kiali/kiali/config"
	"github.com/kiali/kiali/external_deployments/model"
	"github.com/kiali/kiali/models"
	"github.com/kiali/kiali/observability"
)

type DeploymentService struct {
	deploymentClientInterface DeploymentClient
	conf                      *config.Config
}

func NewDeploymentService(conf *config.Config, client DeploymentClient) (*DeploymentService, error) {

	return &DeploymentService{
		deploymentClientInterface: client,
		conf:                      conf,
	}, nil
}

func (in *DeploymentService) client() (DeploymentClient, error) {
	if !in.conf.ExternalServices.ExternalDeployments.Enabled {
		return nil, fmt.Errorf("external deployments are not enabled")
	}
	if in.deploymentClientInterface == nil {
		return nil, fmt.Errorf("external deployments service is not initialized")
	}

	return in.deploymentClientInterface, nil
}
func (in *DeploymentService) ListDeploymentsInRange(ctx context.Context, q models.DeploymentsQuery) ([]*model.Deployment, error) {
	client, err := in.client()
	if err != nil {
		return nil, err
	}

	var end observability.EndFunc
	ctx, end = observability.StartSpan(ctx, "ListDeploymentsInRange",
		observability.Attribute("package", "external_deployments"),
		//observability.Attribute(observability.TracingClusterTag, query.Cluster),
		observability.Attribute("cluster", q.Cluster),
		observability.Attribute("namespace", q.Namespace),
		observability.Attribute("repository", client.GetRepo()),
	)
	defer end()

	deployments, err := client.ListDeploymentsInRange(ctx, q.From, q.To)
	if err != nil {
		return nil, err
	}
	return deployments, nil
}

func (in *DeploymentService) SetRepo(ctx context.Context, repo string) error {
	client, err := in.client()
	if err != nil {
		return err
	}
	err = client.SetRepo(ctx, repo)
	if err != nil {
		return err
	}
	return nil
}
