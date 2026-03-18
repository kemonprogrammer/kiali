package gh

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/google/go-github/v81/github"
	"golang.org/x/sync/errgroup"

	"github.com/kiali/kiali/external_deployments/model"
)

type GithubDeploymentService struct {
	clientInterface       GithubClientInterface
	repo                  string
	ghDeployments         []*github.Deployment
	successfulDeployments []*model.Deployment
}

func NewGithubDeploymentService(clientInterface GithubClientInterface, repo string) (*GithubDeploymentService, error) {
	if clientInterface == nil {
		return nil, fmt.Errorf("clientInterface cannot be nil")
	}
	return &GithubDeploymentService{
		clientInterface: clientInterface,
		repo:            repo,
	}, nil
}

func (gs *GithubDeploymentService) ListDeployments(ctx context.Context) ([]*model.Deployment, error) {
	err := gs.loadDeployments(ctx)

	if err != nil {
		return nil, err
	}
	return toDeployments(gs.ghDeployments), nil
}

func (gs *GithubDeploymentService) ValidateRepo(ctx context.Context) error {
	_, _, err := gs.clientInterface.GetRepository(ctx, gs.repo)
	if err != nil {
		return err
	}
	return nil
}

// loadSuccessfulDeploymentsInRange
//
// Pseudocode:
//
// load deployments
// filter deploys: from after updated at and to is before created at
// if not in successfulDeployments
//   - fetch deployment status
//   - if status successful present put into successfulDeployments
//
// before updating cache sort the deployments by succeededAt
// filter successful deployments in time range
func (gs *GithubDeploymentService) loadSuccessfulDeploymentsInRange(ctx context.Context, from, to time.Time) error {
	if err := gs.loadDeployments(ctx); err != nil {
		return err
	}

	allDeploys := toDeployments(gs.ghDeployments)

	// load successful deployments in Range
	possibleSuccessfulDeploys := filterTimerangeBySuccessPossible(allDeploys, from, to)

	// only refresh success status if not already succeeded
	newPossibleSuccessfulDeploys := make([]*model.Deployment, 0, len(possibleSuccessfulDeploys))
	for _, possibleDeploy := range possibleSuccessfulDeploys {

		if !slices.ContainsFunc(gs.successfulDeployments, func(deploy *model.Deployment) bool {
			return deploy.ID == possibleDeploy.ID
		}) {
			newPossibleSuccessfulDeploys = append(newPossibleSuccessfulDeploys, possibleDeploy)
		}
	}

	populated, err := gs.populateSuccessStatus(ctx, newPossibleSuccessfulDeploys)
	if err != nil {
		return err
	}

	newSuccessfulDeploys := append(gs.successfulDeployments, populated...)
	slices.SortFunc(newSuccessfulDeploys, func(a, b *model.Deployment) int {
		return int(b.SucceededAt.Unix() - a.SucceededAt.Unix()) // assumption: running on 64-bit or higher architecture
	})

	gs.successfulDeployments = newSuccessfulDeploys
	return nil
}

// ListDeploymentsInRange lists deployments with a deployment status successful in range [from, to]
func (gs *GithubDeploymentService) ListDeploymentsInRange(ctx context.Context, from, to time.Time) ([]*model.Deployment, error) {

	err := gs.loadSuccessfulDeploymentsInRange(ctx, from, to)
	if err != nil {
		return nil, err
	}
	successful := gs.successfulDeployments
	fmt.Printf("successful: %d\n", len(successful))

	inRange := filterTimerangeBySucceededAt(successful, from, to)

	var oneBefore *model.Deployment
	for _, sd := range gs.successfulDeployments {
		if sd.SucceededAt.Before(from) {
			oneBefore = sd
			break
		}
	}

	populated, err := gs.populateWithCommits(ctx, append(inRange, oneBefore))
	if err != nil {
		return nil, err
	}

	// remove one before
	if oneBefore != nil {
		populated = populated[:len(populated)-1]
	}
	return populated, nil
}

// loadDeployments loads all deployments on the first time and stores them in cache
func (gs *GithubDeploymentService) loadDeployments(ctx context.Context) error {
	if len(gs.ghDeployments) > 0 {
		prevNewestID := gs.ghDeployments[0].GetID()

		newDeployCount := -1
		var allDeploys []*github.Deployment

		perPage := 100
		opts := &github.DeploymentsListOptions{
			ListOptions: github.ListOptions{Page: 1, PerPage: perPage},
		}

		for opts.ListOptions.Page > 0 && newDeployCount == -1 {
			deploys, resp, err := gs.clientInterface.ListDeployments(ctx, gs.repo, opts)
			if err != nil {
				return fmt.Errorf("error while fetching github ghDeployments: %w", err)
			}

			// search until previous new deployment found
			for i, deploy := range deploys {
				if deploy.GetID() == prevNewestID {
					newDeployCount = i + (opts.ListOptions.Page-1)*perPage
					break
				}
			}
			allDeploys = append(allDeploys, deploys...)

			if resp.Rate.Remaining <= 10 {
				return fmt.Errorf("rate limit nearly exhausted, only 10 calls remaining; resets at %v",
					resp.Rate.Reset)
			}

			opts.ListOptions.Page = resp.NextPage
		}

		if newDeployCount == 0 {
			return nil
		}

		if newDeployCount > 0 {
			// assumption deployments from API are sorted by creation date in descending oder
			newDeploys := allDeploys[:newDeployCount]
			gs.ghDeployments = append(newDeploys, gs.ghDeployments...)
			return nil
		}
		return fmt.Errorf("Could not find cached deployment %d\n", prevNewestID)
	}

	var allDeploys []*github.Deployment
	opts := &github.DeploymentsListOptions{
		ListOptions: github.ListOptions{Page: 1},
	}

	for opts.ListOptions.Page > 0 {
		deploys, resp, err := gs.clientInterface.ListDeployments(ctx, gs.repo, opts)
		if err != nil {
			return fmt.Errorf("error while fetching github ghDeployments: %w", err)
		}

		allDeploys = append(allDeploys, deploys...)

		if resp.Rate.Remaining <= 10 {
			return fmt.Errorf("rate limit nearly exhausted, only 10 calls remaining; resets at %v",
				resp.Rate.Reset)
		}

		opts.ListOptions.Page = resp.NextPage
	}

	gs.ghDeployments = allDeploys
	return nil
}

func (gs *GithubDeploymentService) populateWithCommits(ctx context.Context, deployments []*model.Deployment) ([]*model.Deployment, error) {
	if len(deployments) <= 1 {
		return deployments, nil
	}

	// Create an errgroup with a derived context that cancels if any goroutine errors out.
	g, gCtx := errgroup.WithContext(ctx)

	for i := range len(deployments) - 1 {

		g.Go(func() error {
			d := deployments[i]
			head := deployments[i].SHA
			base := deployments[i+1].SHA

			// Use the gCtx so this request cancels if another goroutine fails
			commitCmp, err := gs.clientInterface.CompareCommits(gCtx, gs.repo, base, head, nil)
			if err != nil {
				return fmt.Errorf("error while comparing commits: %w", err)
			}

			d.ComparisonURL = commitCmp.GetHTMLURL()

			switch status := commitCmp.GetStatus(); status {
			case "ahead":
				d.Added = toCommits(commitCmp)

			case "behind":
				behindCmp, err := gs.clientInterface.CompareCommits(gCtx, gs.repo, head, base, &github.ListOptions{})
				if err != nil {
					return fmt.Errorf("error comparing behind commits: %w", err)
				}
				d.Removed = toCommits(behindCmp)

			case "diverged":
				d.Added = toCommits(commitCmp)
				mergeBase := commitCmp.GetMergeBaseCommit().GetSHA()
				divergedCmp, err := gs.clientInterface.CompareCommits(gCtx, gs.repo, mergeBase, base, &github.ListOptions{})
				if err != nil {
					return fmt.Errorf("error comparing diverged commits: %w", err)
				}
				d.Removed = toCommits(divergedCmp)

			case "identical":
				// No action needed if slices are already nil or empty
			default:
				return fmt.Errorf("unexpected commit status: %s", status)
			}

			return nil
		})
	}

	// Wait blocks until all goroutines finish, returning the first non-nil error (if any)
	if err := g.Wait(); err != nil {
		return nil, err
	}

	return deployments, nil
}

func (gs *GithubDeploymentService) filterSuccessful(ctx context.Context, deployments []*model.Deployment) ([]*model.Deployment, error) {
	successful := make([]*model.Deployment, 0, len(deployments))

	for _, d := range deployments {
		statuses, err := gs.clientInterface.ListDeploymentStatuses(ctx, gs.repo, d.ID, &github.ListOptions{
			PerPage: 10,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to get deployment statuses for %d: %w", d.ID, err)
		}

		for _, status := range statuses {
			if status.GetState() == "success" {
				successful = append(successful, d)
				break
			}
		}
	}
	return successful, nil
}

func filterTimerangeBySucceededAt(deployments []*model.Deployment, from time.Time, to time.Time) []*model.Deployment {
	filtered := make([]*model.Deployment, 0, len(deployments))
	for _, d := range deployments {
		if d.SucceededAt.After(from) && d.SucceededAt.Before(to) {
			filtered = append(filtered, d)
		}
	}
	return filtered
}

// filterTimerangeBySuccessPossible filters deployments which could have a succeeded in the timeframe
func filterTimerangeBySuccessPossible(deployments []*model.Deployment, from time.Time, to time.Time) []*model.Deployment {
	filtered := make([]*model.Deployment, 0, len(deployments))
	for _, d := range deployments {
		if d.UpdatedAt.After(from) && d.CreatedAt.Before(to) {
			filtered = append(filtered, d)
		}
	}
	return filtered
}

// populateSuccessStatus assumption: deployment status states: x -> success -> inactive
func (gs *GithubDeploymentService) populateSuccessStatus(ctx context.Context, deploys []*model.Deployment) ([]*model.Deployment, error) {
	successful := make([]*model.Deployment, 0, len(deploys))

	for _, d := range deploys {
		// todo get all deployment statuses in case there is a next page
		statuses, err := gs.clientInterface.ListDeploymentStatuses(ctx, gs.repo, d.ID, &github.ListOptions{
			PerPage: 30,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to get deployment statuses for %d: %w", d.ID, err)
		}

		for _, status := range statuses {
			if status.GetState() == "success" {
				d.SucceededAt = status.GetUpdatedAt().Time
				successful = append(successful, d)
				break
			}
		}
	}
	return successful, nil
}
