package business

import (
	"fmt"

	"github.com/kiali/kiali/config"
	"github.com/kiali/kiali/external_deployments"
)

type ExternalDeploymentService struct {
	app         *AppService
	conf        *config.Config
	svc         *SvcService
	deployments external_deployments.ClientInterface
	workload    *WorkloadService
}

func NewExternalDeploymentService(conf *config.Config, deployments external_deployments.ClientInterface, svcService *SvcService, workloadService *WorkloadService, appService *AppService) ExternalDeploymentService {
	return ExternalDeploymentService{
		app:         appService,
		conf:        conf,
		svc:         svcService,
		deployments: deployments,
		workload:    workloadService,
	}
}

func (in *ExternalDeploymentService) client() (external_deployments.ClientInterface, error) {
	if !in.conf.ExternalServices.ExternalDeployments.Enabled {
		return nil, fmt.Errorf("external deployments is not enabled")
	}

	if in.deployments == nil {
		return nil, fmt.Errorf("tracing client is not initialized")
	}

	return in.deployments, nil
}
