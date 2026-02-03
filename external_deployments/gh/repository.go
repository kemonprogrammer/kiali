package gh

import (
	"context"
	"fmt"

	"github.com/google/go-github/v81/github"
)

type Repository interface {
	ListDeployments(ctx context.Context, opts *github.DeploymentsListOptions) ([]*github.Deployment, error)
	ListDeploymentStatuses(ctx context.Context, id int64, opts *github.ListOptions) ([]*github.DeploymentStatus, error)
	CompareCommits(ctx context.Context, base, head string, opts *github.ListOptions) (*github.CommitsComparison, error)
}
type GithubRepository struct {
	client                   *github.Client
	repo, owner, environment string
}

func NewGithubRepository(client *github.Client, owner, repo, environment string) (Repository, error) {
	if client == nil {
		return nil, fmt.Errorf("repo cannot be nil")
	}
	return &GithubRepository{
		client:      client,
		owner:       owner,
		repo:        repo,
		environment: environment,
	}, nil
}

func (gc *GithubRepository) ListDeployments(ctx context.Context, opts *github.DeploymentsListOptions) ([]*github.Deployment, error) {
	if opts == nil {
		opts = &github.DeploymentsListOptions{
			Environment: gc.environment,
			//SHA:         "",
			//Ref:         "",
			//Task:        "",
			//ListOptions: github.ListOptions{}, // todo handle more than 30 ghDeployments (default)
		}
	}
	deploys, _, err := gc.client.Repositories.ListDeployments(ctx, gc.owner, gc.repo, opts)
	return deploys, err
}

func (gc *GithubRepository) ListDeploymentStatuses(ctx context.Context, id int64, opts *github.ListOptions) ([]*github.DeploymentStatus, error) {
	deploys, _, err := gc.client.Repositories.ListDeploymentStatuses(ctx, gc.owner, gc.repo, id, opts)
	return deploys, err
}

func (gc *GithubRepository) CompareCommits(ctx context.Context, base, head string, opts *github.ListOptions) (*github.CommitsComparison, error) {
	if opts == nil {
		opts = &github.ListOptions{
			// todo handle more than 30 commits  (default) -> maybe "<first 7 commits> 24 more commits\n<compare-url>"
		}
	}
	deploys, _, err := gc.client.Repositories.CompareCommits(ctx, gc.owner, gc.repo, base, head, opts)
	return deploys, err
}
