package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/google/go-github/v81/github"
	"github.com/gorilla/mux"

	"github.com/kiali/kiali/cache"
	"github.com/kiali/kiali/config"
	"github.com/kiali/kiali/external_deployments"
	"github.com/kiali/kiali/external_deployments/gh"
	"github.com/kiali/kiali/istio"
	"github.com/kiali/kiali/kubernetes"
	"github.com/kiali/kiali/models"
)

// todo move To models/deployments.go or include in models.metrics
type DeploymentsQuery struct {
	From, To                     time.Time
	Cluster, Namespace, Workload string
}

type DeploymentResponse struct {
	Deployments []*external_deployments.Deployment `json:"deployments"`
}

// WorkloadDeployments is the API handler to fetch GitHub deployments, related to a single workload
func WorkloadDeployments(
	conf *config.Config,
	cache cache.KialiCache,
	clientFactory kubernetes.ClientFactory,
	discovery istio.MeshDiscovery,
	extDeploysClientLoader func() external_deployments.ClientInterface,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		namespace := vars["namespace"]
		workload := vars["workload"]
		cluster := clusterNameFromQuery(conf, r.URL.Query())

		// todo refactor: read config at the same location as for tracing
		githubPat := conf.ExternalServices.ExternalDeployments.Auth.Token.String()
		env := conf.ExternalServices.ExternalDeployments.Environment
		ctx := context.Background()
		client := github.NewClient(nil).WithAuthToken(githubPat)

		var owner string
		user, _, err := client.Users.Get(context.Background(), "")
		if err == nil {
			owner = *user.Login
		}

		// todo get from workload
		repo := "github-go-client"

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
