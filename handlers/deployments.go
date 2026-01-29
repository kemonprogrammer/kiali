package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/go-github/v81/github"
	"github.com/gorilla/mux"

	"github.com/kiali/kiali/cache"
	"github.com/kiali/kiali/config"
	"github.com/kiali/kiali/grafana"
	"github.com/kiali/kiali/istio"
	"github.com/kiali/kiali/kubernetes"
	"github.com/kiali/kiali/models"
	"github.com/kiali/kiali/prometheus"
)

// todo move To models/deployments.go or include in models.metrics
type DeploymentsQuery struct {
	From, To                     time.Time
	Cluster, Namespace, Workload string
}

type DeploymentResponse struct {
	deployment github.Deployment
}

func GetTitle(message string) string {
	return strings.Split(message, "\n")[0]
}

// WorkloadDeployments is the API handler To fetch GitHub deployments, related To a single workload
func WorkloadDeployments(
	conf *config.Config,
	cache cache.KialiCache,
	clientFactory kubernetes.ClientFactory,
	discovery istio.MeshDiscovery,
	prom prometheus.ClientInterface,
	grafana *grafana.Service,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// todo move To server.go
		owner := os.Getenv("OWNER")
		repo := "github-go-client"
		githubPat := os.Getenv("GITHUB_PAT")
		env := os.Getenv("ENVIRONMENT")
		ctx := context.Background()
		client := github.NewClient(nil).WithAuthToken(githubPat)
		// todo end

		vars := mux.Vars(r)
		namespace := vars["namespace"]
		workload := vars["workload"]
		cluster := clusterNameFromQuery(conf, r.URL.Query())

		// todo check what this does
		_, err := checkNamespaceAccess(w, r, conf, cache, discovery, clientFactory, namespace, cluster)
		if err != nil {
			return
		}
		//models.IstioMetricsQuery{}
		params := DeploymentsQuery{Cluster: cluster, Namespace: namespace, Workload: workload}

		if err := extractDeploymentQueryParams(r, &params, nil); err != nil {
			RespondWithError(w, http.StatusBadRequest, err.Error())
			return
		}

		// logic
		deploymentsAll, _, err := client.Repositories.ListDeployments(ctx, owner, repo, &github.DeploymentsListOptions{
			SHA:         "",
			Ref:         "",
			Task:        "",
			Environment: env,
			ListOptions: github.ListOptions{}, // todo handle more than 30 deployments (default)
		})
		if err != nil {
			RespondWithError(w, http.StatusServiceUnavailable, err.Error())
			return
		}

		//deployments = FilterTimerange(deployments, params.Start, params.End)
		deployments := FilterTimerange(deploymentsAll, params.From, params.To)
		//fmt.Printf("len deploys: %d", len(deployments))

		for i, d := range deployments {
			fmt.Printf("\nDeployment %d:\n", d.GetID())

			fmt.Printf("sha: %s\n", d.GetSHA())
			fmt.Printf("created at: %s\n", d.GetCreatedAt())

			if len(deployments) < 2 || i+1 >= len(deployments) {
				continue
			}
			head := deployments[i].GetSHA()
			base := deployments[i+1].GetSHA()

			commitCmp, _, err := client.Repositories.CompareCommits(ctx, owner, repo, base, head, &github.ListOptions{
				Page:    0,
				PerPage: 10, // todo handle more than 10 commits -> maybe "61 more commits\n<compare-url>"
			})

			if err != nil {
				fmt.Println(err)
				return
			}

			//fmt.Printf("Head and base differ by %d commits:\n", commitCmp.GetTotalCommits())
			for _, c := range commitCmp.Commits {
				fmt.Printf("+ %s\n", GetTitle(c.Commit.GetMessage()))
			}
		}

		// // leave it out for now to save github api calls
		//successfulDeployments, err := FilterSuccessful(client, ctx, owner, repo, deployments)
		//if err != nil {
		//	fmt.Println(err)
		//}

		response := map[string]interface{}{"params": params, "deployments": deployments}
		//RespondWithJSON(w, http.StatusOK, deployments)
		RespondWithJSON(w, http.StatusOK, response)
		//if err := extractIstioMetricsQueryParams(r, &params, namespaceInfo); err != nil {
		//	RespondWithError(w, http.StatusBadRequest, err.Error())
		//	return
		//}
		//
		//metricsService := business.NewMetricsService(prom, conf)
		//metrics, err := metricsService.GetMetrics(r.Context(), params, business.GetIstioScaler())
		//if err != nil {
		//	RespondWithError(w, http.StatusServiceUnavailable, err.Error())
		//	return
		//}
		//dashboard := business.NewDashboardsService(conf, grafana, prom, namespaceInfo, nil).BuildIstioDashboard(metrics, params.Direction)
		//RespondWithJSON(w, http.StatusOK, dashboard)
	}
}

func FilterTimerange(deployments []*github.Deployment, from time.Time, to time.Time) []*github.Deployment {
	filteredDeploys := make([]*github.Deployment, 0, len(deployments))
	for _, d := range deployments {
		if d.CreatedAt.Time.After(from) && d.CreatedAt.Time.Before(to) {
			filteredDeploys = append(filteredDeploys, d)
		}
	}
	return filteredDeploys
}

func extractDeploymentQueryParams(r *http.Request, query *DeploymentsQuery, namespaceInfo *models.Namespace) error {
	queryParams := r.URL.Query()
	query.To = time.Now()

	if dur := queryParams.Get("duration"); dur != "" {
		if num, err := strconv.ParseInt(dur, 10, 64); err == nil {
			duration := time.Duration(num) * time.Second
			query.From = query.To.Add(-duration)
		} else {
			return errors.New("bad request, cannot parse query parameter 'from'")
		}
	}

	return nil
}
