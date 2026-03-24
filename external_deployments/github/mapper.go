package github

import (
	"strings"

	"github.com/google/go-github/v81/github"

	"github.com/kiali/kiali/external_deployments/model"
)

func toCommits(commitCmp *github.CommitsComparison) []*model.Commit {
	commits := make([]*model.Commit, commitCmp.GetTotalCommits())
	for i, commit := range commitCmp.Commits {
		commits[i] = toCommit(commit)
	}
	return commits
}

func toDeployment(ghDeploy *github.Deployment) *model.Deployment {
	if ghDeploy == nil {
		return nil
	}
	return &model.Deployment{
		ID:            ghDeploy.GetID(),
		SHA:           ghDeploy.GetSHA(),
		CreatedAt:     ghDeploy.GetCreatedAt().Time,
		UpdatedAt:     ghDeploy.GetUpdatedAt().Time,
		ComparisonURL: "",
		Added:         []*model.Commit{},
		Removed:       []*model.Commit{},
	}
}

func toDeployments(ghDeployments []*github.Deployment) []*model.Deployment {
	deployments := make([]*model.Deployment, len(ghDeployments))
	for i, d := range ghDeployments {
		deployments[i] = toDeployment(d)
	}
	return deployments
}

func toCommit(commit *github.RepositoryCommit) *model.Commit {
	return &model.Commit{
		SHA:   commit.GetSHA(), // sha somehow stored in commit, not commit.Commit
		Title: ParseCommitTitle(commit.Commit.GetMessage()),
		URL:   commit.GetHTMLURL(),
	}
}

func ParseCommitTitle(message string) string {
	title, _, _ := strings.Cut(message, "\n")
	return title
}
