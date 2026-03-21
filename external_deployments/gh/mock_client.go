package gh

import (
	"context"
	"fmt"
	"time"

	"github.com/google/go-github/v81/github"
)

type MockGithubClient struct {
	owner, environment string
}

func NewMockGithubClient() GithubClientInterface {
	return &MockGithubClient{
		owner:       "mock-owner",
		environment: "mock-environment",
	}
}

func (gc *MockGithubClient) GetRepository(_ context.Context, repoName string) (*github.Repository, *github.Response, error) {
	time.Sleep(500 * time.Millisecond)

	repo := &github.Repository{
		ID:          github.Ptr(int64(123456)),
		Name:        github.Ptr(repoName),
		FullName:    github.Ptr(gc.owner + "/" + repoName),
		Description: github.Ptr("This is a mocked repository for testing"),
		HTMLURL:     github.Ptr("https://github.com/" + gc.owner + "/" + repoName),
		Private:     github.Ptr(true),
	}

	return repo, &github.Response{NextPage: 0}, nil
}

func (gc *MockGithubClient) ListDeployments(_ context.Context, _ string, _ *github.DeploymentsListOptions) ([]*github.Deployment, *github.Response, error) {
	time.Sleep(400 * time.Millisecond)

	length := 1000
	deploys := make([]*github.Deployment, 0, length)

	for i := 0; i < length; i++ {
		deploys = append(deploys, &github.Deployment{
			// Using int64() cast so the generic Ptr infers *int64 instead of *int
			ID:          github.Ptr(int64(1001 + i)),
			SHA:         github.Ptr("def456ghi789"),
			Ref:         github.Ptr("main"),
			Task:        github.Ptr("deploy"),
			Environment: github.Ptr("production"),
			Description: github.Ptr(fmt.Sprintf("Mocked production deployment %d", i+1)),
			Creator: &github.User{
				Login: github.Ptr("octocat"),
			},
			CreatedAt: &github.Timestamp{Time: time.Unix(0, 0)},
			UpdatedAt: &github.Timestamp{Time: time.Now()},
		})
	}

	return deploys, &github.Response{NextPage: 0, Rate: github.Rate{Remaining: 1000}}, nil
}

func (gc *MockGithubClient) ListDeploymentStatuses(_ context.Context, _ string, _ int64, _ *github.ListOptions) ([]*github.DeploymentStatus, *github.Response, error) {
	time.Sleep(300 * time.Millisecond)

	statuses := []*github.DeploymentStatus{
		{
			ID:             github.Ptr(int64(2001)),
			State:          github.Ptr("success"),
			Description:    github.Ptr("Deployment finished successfully"),
			EnvironmentURL: github.Ptr("https://prod.example.com"),
			LogURL:         github.Ptr("https://ci.example.com/logs/1"),
			Creator: &github.User{
				Login: github.Ptr("octocat"),
			},
			CreatedAt: &github.Timestamp{Time: time.Now()},
			UpdatedAt: &github.Timestamp{Time: time.Now().Add(-5 * time.Minute)},
		},
	}

	return statuses, &github.Response{NextPage: 0, Rate: github.Rate{Remaining: 1000}}, nil
}

func (gc *MockGithubClient) CompareCommits(_ context.Context, _, _, _ string, _ *github.ListOptions) (*github.CommitsComparison, error) {
	time.Sleep(500 * time.Millisecond)

	commitCmp := &github.CommitsComparison{
		HTMLURL:      github.Ptr("https://example.com"),
		Status:       github.Ptr("ahead"),
		AheadBy:      github.Ptr(2),
		BehindBy:     github.Ptr(0),
		TotalCommits: github.Ptr(2),
		BaseCommit: &github.RepositoryCommit{
			SHA: github.Ptr("abc123def456"),
		},
		Commits: []*github.RepositoryCommit{
			{
				SHA: github.Ptr("def456ghi789"),
				Commit: &github.Commit{
					Message: github.Ptr("feat: mocked commit 1"),
					Author: &github.CommitAuthor{
						Name:  github.Ptr("Octo Cat"),
						Email: github.Ptr("octocat@github.com"),
					},
				},
			},
			{
				SHA: github.Ptr("ghi789jkl012"),
				Commit: &github.Commit{
					Message: github.Ptr("fix: mocked commit 2"),
					Author: &github.CommitAuthor{
						Name:  github.Ptr("Octo Cat"),
						Email: github.Ptr("octocat@github.com"),
					},
				},
			},
		},
		MergeBaseCommit: &github.RepositoryCommit{
			SHA: github.Ptr("def456ghi789"),
			Commit: &github.Commit{
				Message: github.Ptr("feat: mocked commit 1"),
				Author: &github.CommitAuthor{
					Name:  github.Ptr("Octo Cat"),
					Email: github.Ptr("octocat@github.com"),
				},
			},
		},
	}

	return commitCmp, nil
}
