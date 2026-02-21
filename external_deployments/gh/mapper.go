package gh

import (
	"strings"

	"github.com/google/go-github/v81/github"

	"github.com/kiali/kiali/external_deployments"
)

func toCommits(commitCmp *github.CommitsComparison) []*external_deployments.Commit {
	commits := make([]*external_deployments.Commit, commitCmp.GetTotalCommits())
	for i, commit := range commitCmp.Commits {
		commits[i] = toCommit(commit)
	}
	return commits
}

func toDeployment(ghDeploy *github.Deployment) *external_deployments.Deployment {
	if ghDeploy == nil {
		return nil
	}
	return &external_deployments.Deployment{
		ID:            ghDeploy.GetID(),
		URL:           ghDeploy.GetURL(),
		SHA:           ghDeploy.GetSHA(),
		CreatedAt:     ghDeploy.GetCreatedAt().Time,
		UpdatedAt:     ghDeploy.GetUpdatedAt().Time,
		ComparisonURL: "",
		Added:         make([]*external_deployments.Commit, 0),
		Removed:       make([]*external_deployments.Commit, 0),
	}
}

func toDeployments(ghDeployments []*github.Deployment) []*external_deployments.Deployment {
	deployments := make([]*external_deployments.Deployment, len(ghDeployments))
	for i, d := range ghDeployments {
		deployments[i] = toDeployment(d)
	}
	return deployments
}

func toCommit(commit *github.RepositoryCommit) *external_deployments.Commit {
	return &external_deployments.Commit{
		SHA:   commit.GetSHA(), // sha somehow stored in commit, not commit.Commit
		Title: ParseCommitTitle(commit.Commit.GetMessage()),
		URL:   commit.Commit.GetURL(),
	}
}

func ParseCommitTitle(message string) string {
	title, _, _ := strings.Cut(message, "\n")
	return title
}
