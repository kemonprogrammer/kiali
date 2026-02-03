package external_deployments

import (
	"context"
	"time"
)

type ClientInterface interface {
	ListDeployments(ctx context.Context) ([]*Deployment, error)
	ListDeploymentsInRange(ctx context.Context, from, to time.Time) ([]*Deployment, error)
}
