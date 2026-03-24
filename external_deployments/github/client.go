package github

import (
	"context"
	"fmt"
	"log"
	"slices"
	"sync"
	"time"

	"github.com/google/go-github/v81/github"
	"golang.org/x/sync/errgroup"

	"github.com/kiali/kiali/external_deployments/model"
)

type DeploymentClient struct {
	api                   API
	repo                  string
	ghDeployments         []*github.Deployment
	successfulDeployments []*model.Deployment
}

func NewDeploymentClient(api API) (*DeploymentClient, error) {
	if api == nil {
		return nil, fmt.Errorf("api cannot be nil")
	}
	return &DeploymentClient{
		api: api,
	}, nil
}

func (gdc *DeploymentClient) ListDeployments(ctx context.Context) ([]*model.Deployment, error) {
	err := gdc.loadDeployments(ctx)

	if err != nil {
		return nil, err
	}
	return toDeployments(gdc.ghDeployments), nil
}

// ListDeploymentsInRange lists deployments with a deployment status successful in range [from, to]
func (gdc *DeploymentClient) ListDeploymentsInRange(ctx context.Context, from, to time.Time) ([]*model.Deployment, error) {

	err := gdc.loadSuccessfulDeploymentsInRange(ctx, from, to)
	if err != nil {
		return nil, err
	}
	successful := gdc.successfulDeployments

	inRange := filterTimerangeBySucceededAt(successful, from, to)

	var oneBefore *model.Deployment
	for _, sd := range gdc.successfulDeployments {
		if sd.SucceededAt.Before(from) {
			oneBefore = sd
			break
		}
	}

	if oneBefore != nil {
		inRange = append(inRange, oneBefore)
	}

	populated, err := gdc.populateWithCommits(ctx, inRange)
	if err != nil {
		return nil, err
	}

	// remove one before
	if oneBefore != nil {
		populated = populated[:len(populated)-1]
	}
	return populated, nil
}

func (gdc *DeploymentClient) SetRepo(ctx context.Context, repo string) error {
	_, _, err := gdc.api.GetRepository(ctx, repo)
	if err != nil {
		return err
	}
	gdc.repo = repo
	return nil
}

func (gdc *DeploymentClient) GetRepo() string {
	return gdc.repo
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
func (gdc *DeploymentClient) loadSuccessfulDeploymentsInRange(ctx context.Context, from, to time.Time) error {
	if err := gdc.loadDeployments(ctx); err != nil {
		return err
	}

	allDeploys := toDeployments(gdc.ghDeployments)

	// load successful deployments in Range
	possibleSuccessfulDeploys := filterTimerangeBySuccessPossible(allDeploys, from, to)

	// only refresh success status if not already succeeded
	newPossibleSuccessfulDeploys := make([]*model.Deployment, 0, len(possibleSuccessfulDeploys))
	for _, possibleDeploy := range possibleSuccessfulDeploys {

		if !slices.ContainsFunc(gdc.successfulDeployments, func(deploy *model.Deployment) bool {
			return deploy.ID == possibleDeploy.ID
		}) {
			newPossibleSuccessfulDeploys = append(newPossibleSuccessfulDeploys, possibleDeploy)
		}
	}

	populated, err := gdc.populateSuccessStatus(ctx, newPossibleSuccessfulDeploys)
	if err != nil {
		return err
	}

	newSuccessfulDeploys := append(gdc.successfulDeployments, populated...)
	slices.SortFunc(newSuccessfulDeploys, func(a, b *model.Deployment) int {
		return int(b.SucceededAt.Unix() - a.SucceededAt.Unix()) // assumption: running on 64-bit or higher architecture
	})

	gdc.successfulDeployments = newSuccessfulDeploys
	return nil
}

// loadDeployments loads all deployments on the first time and stores them in cache
func (gdc *DeploymentClient) loadDeployments(ctx context.Context) error {
	if len(gdc.ghDeployments) > 0 {
		prevNewestID := gdc.ghDeployments[0].GetID()

		newDeployCount := -1
		var allDeploys []*github.Deployment

		perPage := 100
		opts := &github.DeploymentsListOptions{
			ListOptions: github.ListOptions{Page: 1, PerPage: perPage},
		}

		for opts.ListOptions.Page > 0 && newDeployCount == -1 {
			deploys, resp, err := gdc.api.ListDeployments(ctx, gdc.repo, opts)
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
			gdc.ghDeployments = append(newDeploys, gdc.ghDeployments...)
			return nil
		}
		return fmt.Errorf("Could not find cached deployment %d\n", prevNewestID)
	}

	var allDeploys []*github.Deployment
	opts := &github.DeploymentsListOptions{
		ListOptions: github.ListOptions{Page: 1},
	}

	for opts.ListOptions.Page > 0 {
		deploys, resp, err := gdc.api.ListDeployments(ctx, gdc.repo, opts)
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

	gdc.ghDeployments = allDeploys
	return nil
}

func (gdc *DeploymentClient) populateWithCommits(ctx context.Context, deployments []*model.Deployment) ([]*model.Deployment, error) {
	if len(deployments) <= 1 {
		return deployments, nil
	}

	// Create an errgroup with a derived context that cancels if any goroutine errors out.
	g, gCtx := errgroup.WithContext(ctx)
	start := time.Now()

	for i := range len(deployments) - 1 {

		g.Go(func() error {
			d := deployments[i]
			head := deployments[i].SHA
			base := deployments[i+1].SHA

			// Use the gCtx so this request cancels if another goroutine fails
			commitCmp, err := gdc.api.CompareCommits(gCtx, gdc.repo, base, head, nil)
			if err != nil {
				return fmt.Errorf("error while comparing commits: %w", err)
			}

			d.ComparisonURL = commitCmp.GetHTMLURL()

			switch status := commitCmp.GetStatus(); status {
			case "ahead":
				d.Added = toCommits(commitCmp)

			case "behind":
				behindCmp, err := gdc.api.CompareCommits(gCtx, gdc.repo, head, base, &github.ListOptions{})
				if err != nil {
					return fmt.Errorf("error comparing behind commits: %w", err)
				}
				d.Removed = toCommits(behindCmp)

			case "diverged":
				d.Added = toCommits(commitCmp)
				mergeBase := commitCmp.GetMergeBaseCommit().GetSHA()
				divergedCmp, err := gdc.api.CompareCommits(gCtx, gdc.repo, mergeBase, base, &github.ListOptions{})
				if err != nil {
					return fmt.Errorf("error comparing diverged commits: %w", err)
				}
				d.Removed = toCommits(divergedCmp)

			case "identical":
				// No action needed if slices are already nil or empty
			default:
				return fmt.Errorf("unexpected commit status: %s", status)
			}

			return nil // Return nil to signal success to the errgroup
		})
	}

	// Wait blocks until all goroutines finish, returning the first non-nil error (if any)
	if err := g.Wait(); err != nil {
		return nil, err
	}
	log.Printf("TRACE comparing %d times took %v\n", len(deployments)-1, time.Since(start))

	// Sort the slice in place
	slices.SortFunc(deployments, func(a, b *model.Deployment) int {
		return int(b.SucceededAt.Unix() - a.SucceededAt.Unix())
	})

	return deployments, nil
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
func (gdc *DeploymentClient) populateSuccessStatus(ctx context.Context, deploys []*model.Deployment) ([]*model.Deployment, error) {
	successful := make([]*model.Deployment, 0, len(deploys))
	var mu sync.Mutex
	g, gCtx := errgroup.WithContext(ctx)

	start := time.Now()

	for _, d := range deploys {
		g.Go(func() error {
			opts := &github.ListOptions{
				Page: 1,
			}
		out:
			for opts.Page > 0 {
				statuses, resp, err := gdc.api.ListDeploymentStatuses(gCtx, gdc.repo, d.ID, opts)
				if err != nil {
					return fmt.Errorf("failed to get deployment statuses for %d: %w", d.ID, err)
				}
				opts.Page = resp.NextPage

				if resp.Rate.Remaining <= 10 {
					return fmt.Errorf("rate limit nearly exhausted, only 10 calls remaining; resets at %v", resp.Rate.Reset)
				}

				for _, status := range statuses {
					if status.GetState() == "success" {
						d.SucceededAt = status.GetUpdatedAt().Time
						mu.Lock()
						successful = append(successful, d)
						mu.Unlock()
						break out
					}
				}
			}
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}
	log.Printf("TRACE loading success statuses took %v\n", time.Since(start))

	slices.SortFunc(successful, func(a, b *model.Deployment) int {
		return int(b.SucceededAt.Unix() - a.SucceededAt.Unix())
	})
	return successful, nil
}
