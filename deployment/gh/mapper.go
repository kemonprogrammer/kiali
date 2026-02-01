package gh

import (
	"strings"

	"github.com/google/go-github/v81/github"

	"github.com/kiali/kiali/deployment"
)

func toCommits(commitCmp *github.CommitsComparison) []*deployment.Commit {
	commits := make([]*deployment.Commit, commitCmp.GetTotalCommits())
	for i, commit := range commitCmp.Commits {
		commits[i] = toCommit(commit)
	}
	return commits
}

func toDeployment(ghDeploy *github.Deployment) *deployment.Deployment {
	if ghDeploy == nil {
		return nil
	}
	return &deployment.Deployment{
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

func toDeployments(ghDeployments []*github.Deployment) []*deployment.Deployment {
	deployments := make([]*deployment.Deployment, len(ghDeployments))
	for i, d := range ghDeployments {
		deployments[i] = toDeployment(d)
	}
	return deployments
}

func toCommit(commit *github.RepositoryCommit) *deployment.Commit {
	return &deployment.Commit{
		SHA:   commit.GetSHA(), // sha somehow stored in commit, not commit.Commit
		Title: ParseCommitTitle(commit.Commit.GetMessage()),
		URL:   commit.Commit.GetURL(),
	}
}

func ParseCommitTitle(message string) string {
	title, _, _ := strings.Cut(message, "\n")
	return title
}
