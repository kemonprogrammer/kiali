package gh

import (
	"context"
	"fmt"
	"time"

	"github.com/google/go-github/v81/github"

	"github.com/kiali/kiali/config"
	"github.com/kiali/kiali/log"
)

// GithubClientInterface mock for testing
type GithubClientInterface interface {
	GetRepository(ctx context.Context, repoName string) (*github.Repository, *github.Response, error)
	ListDeployments(ctx context.Context, repoName string, opts *github.DeploymentsListOptions) ([]*github.Deployment, *github.Response, error)
	ListDeploymentStatuses(ctx context.Context, repoName string, id int64, opts *github.ListOptions) ([]*github.DeploymentStatus, *github.Response, error)
	CompareCommits(ctx context.Context, repoName, base, head string, opts *github.ListOptions) (*github.CommitsComparison, error)
}

type GithubClient struct {
	client             *github.Client
	owner, environment string
}

func MakeGithubClientInterface(conf *config.Config) (GithubClientInterface, error) {
	owner := conf.ExternalServices.ExternalDeployments.Auth.Username.String()
	env := conf.ExternalServices.ExternalDeployments.Environment
	githubPat := conf.ExternalServices.ExternalDeployments.Auth.Token.String()

	if len(env) == 0 {
		env = "production"
	}
	if len(githubPat) == 0 {
		return nil, fmt.Errorf("no external deployments auth token provided")
	}

	//todo remove
	fmt.Printf("Github PAT from config: %s\n", githubPat)
	fmt.Printf("owner from config: %s\n", owner)
	fmt.Printf("env from config: %s\n", env)

	gh := github.NewClient(nil).WithAuthToken(githubPat)
	clientInterface, err := NewGithubClient(gh, owner, env)
	if err != nil {
		return nil, err
	}

	return clientInterface, nil
}

func NewGithubClient(client *github.Client, owner, environment string) (GithubClientInterface, error) {
	if client == nil {
		return nil, fmt.Errorf("github client cannot be nil")
	}
	return &GithubClient{
		client:      client,
		owner:       owner,
		environment: environment,
	}, nil
}

func (gc *GithubClient) GetRepository(ctx context.Context, repoName string) (*github.Repository, *github.Response, error) {
	start := time.Now()
	defer func() {
		log.Tracef("getRepository took %v\n", time.Since(start))
	}()
	repo, resp, err := gc.client.Repositories.Get(ctx, gc.owner, repoName)
	if err != nil {
		return nil, nil, err
	}
	return repo, resp, nil
}

func (gc *GithubClient) ListDeployments(ctx context.Context, repoName string, opts *github.DeploymentsListOptions) ([]*github.Deployment, *github.Response, error) {
	start := time.Now()
	defer func() {
		log.Tracef("")
		log.Tracef("listDeployments took %v\n", time.Since(start))
	}()
	if opts.Environment == "" {
		opts.Environment = gc.environment
	}
	deploys, resp, err := gc.client.Repositories.ListDeployments(ctx, gc.owner, repoName, opts)
	return deploys, resp, err
}

func (gc *GithubClient) ListDeploymentStatuses(ctx context.Context, repoName string, id int64, opts *github.ListOptions) ([]*github.DeploymentStatus, *github.Response, error) {
	start := time.Now()
	defer func() {
		log.Tracef("listDeploymentStatuses took %v\n", time.Since(start))
	}()
	statuses, resp, err := gc.client.Repositories.ListDeploymentStatuses(ctx, gc.owner, repoName, id, opts)
	return statuses, resp, err
}

func (gc *GithubClient) CompareCommits(ctx context.Context, repoName, base, head string, opts *github.ListOptions) (*github.CommitsComparison, error) {
	start := time.Now()
	defer func() {
		log.Tracef("compareCommits took %v\n", time.Since(start))
	}()
	commitCmp, _, err := gc.client.Repositories.CompareCommits(ctx, gc.owner, repoName, base, head, opts)
	if err != nil {
		return nil, err
	}

	return commitCmp, err
}
