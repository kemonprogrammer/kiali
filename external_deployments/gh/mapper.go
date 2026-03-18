package gh

import (
	"strings"

	"github.com/google/go-github/v81/github"

	"github.com/kiali/kiali/external_deployments/types"
)

func toCommits(commitCmp *github.CommitsComparison) []*types.Commit {
	commits := make([]*types.Commit, commitCmp.GetTotalCommits())
	for i, commit := range commitCmp.Commits {
		commits[i] = toCommit(commit)
	}
	return commits
}

func toDeployment(ghDeploy *github.Deployment) *types.Deployment {
	if ghDeploy == nil {
		return nil
	}
	return &types.Deployment{
		ID:            ghDeploy.GetID(),
		URL:           ghDeploy.GetURL(),
		SHA:           ghDeploy.GetSHA(),
		CreatedAt:     ghDeploy.GetCreatedAt().Time,
		UpdatedAt:     ghDeploy.GetUpdatedAt().Time,
		ComparisonURL: "",
		Added:         nil,
		Removed:       nil,
	}
}

func toDeployments(ghDeployments []*github.Deployment) []*types.Deployment {
	deployments := make([]*types.Deployment, len(ghDeployments))
	for i, d := range ghDeployments {
		deployments[i] = toDeployment(d)
	}
	return deployments
}

func toCommit(commit *github.RepositoryCommit) *types.Commit {
	return &types.Commit{
		SHA:   commit.GetSHA(), // sha somehow stored in commit, not commit.Commit
		Title: ParseCommitTitle(commit.Commit.GetMessage()),
		URL:   commit.Commit.GetURL(),
	}
}

func ParseCommitTitle(message string) string {
	title, _, _ := strings.Cut(message, "\n")
	return title
}
