package gh

import (
	"context"
	"fmt"

	"github.com/google/go-github/v81/github"
)

type Repository interface {
	ListDeployments(ctx context.Context, opts *github.DeploymentsListOptions) ([]*github.Deployment, *github.Response, error)
	ListDeploymentStatuses(ctx context.Context, id int64, opts *github.ListOptions) ([]*github.DeploymentStatus, error)
	CompareCommits(ctx context.Context, base, head string, opts *github.ListOptions) (*github.CommitsComparison, error)
}

type GithubRepository struct {
	client                   *github.Client
	name, owner, environment string
	commitCache              map[compareKey]*github.CommitsComparison
}

type compareKey struct {
	Base string
	Head string
}

func NewGithubRepository(client *github.Client, owner, name, environment string) (Repository, error) {
	if client == nil {
		return nil, fmt.Errorf("github client cannot be nil")
	}
	return &GithubRepository{
		client:      client,
		owner:       owner,
		name:        name,
		environment: environment,
		commitCache: make(map[compareKey]*github.CommitsComparison),
	}, nil
}

func (gc *GithubRepository) ListDeployments(ctx context.Context, opts *github.DeploymentsListOptions) ([]*github.Deployment, *github.Response, error) {
	fmt.Printf("TRACE ListDeployments\n")
	if opts.Environment == "" {
		opts.Environment = gc.environment
	}
	deploys, resp, err := gc.client.Repositories.ListDeployments(ctx, gc.owner, gc.name, opts)
	return deploys, resp, err
}

func (gc *GithubRepository) ListDeploymentStatuses(ctx context.Context, id int64, opts *github.ListOptions) ([]*github.DeploymentStatus, error) {
	fmt.Printf("TRACE ListDeploymentStatuses\n")
	statuses, _, err := gc.client.Repositories.ListDeploymentStatuses(ctx, gc.owner, gc.name, id, opts)
	return statuses, err
}

func (gc *GithubRepository) CompareCommits(ctx context.Context, base, head string, opts *github.ListOptions) (*github.CommitsComparison, error) {
	key := compareKey{Base: base, Head: head}
	if res, found := gc.commitCache[key]; found {
		return res, nil
	}

	fmt.Printf("TRACE CompareCommits\n")
	commitCmp, _, err := gc.client.Repositories.CompareCommits(ctx, gc.owner, gc.name, base, head, opts)
	if err != nil {
		return nil, err
	}

	gc.commitCache[key] = commitCmp
	return commitCmp, err
}
