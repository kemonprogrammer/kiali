package handlers

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/google/go-github/v81/github"
	"github.com/gorilla/mux"

	"github.com/kiali/kiali/cache"
	"github.com/kiali/kiali/config"
	"github.com/kiali/kiali/deployment"
	"github.com/kiali/kiali/deployment/gh"
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
	Deployments []*deployment.Deployment `json:"deployments"`
}

// WorkloadDeployments is the API handler to fetch GitHub deployments, related to a single workload
func WorkloadDeployments(
	conf *config.Config,
	cache cache.KialiCache,
	clientFactory kubernetes.ClientFactory,
	discovery istio.MeshDiscovery,
	prom prometheus.ClientInterface,
	grafana *grafana.Service,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// todo get from config
		owner := os.Getenv("OWNER")
		repo := "github-go-client"
		githubPat := os.Getenv("GITHUB_PAT")
		env := os.Getenv("ENVIRONMENT")
		ctx := context.Background()
		client := github.NewClient(nil).WithAuthToken(githubPat)

		// todo move to server.go
		ghRepo, err := gh.NewGithubRepository(client, owner, repo, env)
		if err != nil {
			RespondWithError(w, http.StatusServiceUnavailable, err.Error())
			return
		}

		deploymentService, err := gh.NewService(ghRepo)
		if err != nil {
			RespondWithError(w, http.StatusServiceUnavailable, err.Error())
			return
		}

		// todo end

		vars := mux.Vars(r)
		namespace := vars["namespace"]
		workload := vars["workload"]
		cluster := clusterNameFromQuery(conf, r.URL.Query())

		// todo check what this does
		_, err = checkNamespaceAccess(w, r, conf, cache, discovery, clientFactory, namespace, cluster)
		if err != nil {
			RespondWithError(w, http.StatusServiceUnavailable, err.Error())
			return
		}
		//models.IstioMetricsQuery{}
		params := DeploymentsQuery{Cluster: cluster, Namespace: namespace, Workload: workload}

		if err := extractDeploymentQueryParams(r, &params, nil); err != nil {
			RespondWithError(w, http.StatusBadRequest, err.Error())
			return
		}

		deployments, err := deploymentService.ListDeploymentsInRange(ctx, params.From, params.To)
		fmt.Printf("deployments %+v\n", deployments) // todo remove
		if err != nil {
			RespondWithError(w, http.StatusServiceUnavailable, err.Error())
			return
		}

		response := &DeploymentResponse{Deployments: deployments}
		fmt.Printf("response %+v\n", response) // todo remove
		//RespondWithJSON(w, http.StatusOK, &deployments)
		RespondWithJSON(w, http.StatusOK, response)

		// -- OTHER HANDLER CODE --

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

func extractDeploymentQueryParams(r *http.Request, query *DeploymentsQuery, namespaceInfo *models.Namespace) error {
	queryParams := r.URL.Query()
	query.To = time.Now()

	if to := queryParams.Get("queryTime"); to != "" {
		if num, err := strconv.ParseInt(to, 10, 64); err == nil {
			query.To = time.Unix(num, 0)
		} else {
			return fmt.Errorf("bad request, cannot parse query parameter 'queryTime': %s", to)
		}
	}

	if dur := queryParams.Get("duration"); dur != "" {
		if num, err := strconv.ParseInt(dur, 10, 64); err == nil {
			duration := time.Duration(num) * time.Second
			query.From = query.To.Add(-duration)
		} else {
			return fmt.Errorf("bad request, cannot parse query parameter 'duration': %s", dur)
		}
	}

	return nil
}
