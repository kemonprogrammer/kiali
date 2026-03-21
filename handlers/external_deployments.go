package handlers

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/gorilla/mux"

	"github.com/kiali/kiali/cache"
	"github.com/kiali/kiali/config"
	"github.com/kiali/kiali/external_deployments"
	"github.com/kiali/kiali/external_deployments/model"
	"github.com/kiali/kiali/istio"
	"github.com/kiali/kiali/kubernetes"
	"github.com/kiali/kiali/log"
	"github.com/kiali/kiali/models"
)

type DeploymentsQuery struct {
	From, To                     time.Time
	Cluster, Namespace, Workload string
}

type DeploymentResponse struct {
	Deployments []*model.Deployment `json:"deployments"`
	Total       int                 `json:"total"`
}

// ExternalDeployments is the API handler to fetch GitHub deployments, related to a single workload
func ExternalDeployments(
	conf *config.Config,
	cache cache.KialiCache,
	clientFactory kubernetes.ClientFactory,
	discovery istio.MeshDiscovery,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		namespace := vars["namespace"]
		workload := vars["workload"]
		if len(workload) == 0 {
			workload = vars["app"]
		}
		if len(workload) == 0 {
			workload = vars["service"]
		}
		if len(workload) == 0 {
			RespondWithError(w, http.StatusBadRequest, "No workload provided!")
			return
		}

		cluster := clusterNameFromQuery(conf, r.URL.Query())
		ctx := context.Background()

		owner := conf.ExternalServices.ExternalDeployments.Auth.Username.String()
		if len(owner) == 0 {
			RespondWithError(w, http.StatusBadRequest, "external_deployments.auth.username not set in config")
			return
		}

		repo := extractRepoName(workload)

		deploymentService, err := external_deployments.NewDeploymentService(conf, repo)
		if err != nil {
			RespondWithError(w, http.StatusServiceUnavailable, "could not set up external deployments")
			return
		}

		// params
		if err := deploymentService.ValidateRepo(ctx); err != nil {
			RespondWithError(w, http.StatusServiceUnavailable, fmt.Sprintf("no repository found for workload %s %s", workload, err))
			return
		}

		_, err = checkNamespaceAccess(w, r, conf, cache, discovery, clientFactory, namespace, cluster)
		if err != nil {
			RespondWithError(w, http.StatusServiceUnavailable, err.Error())
			return
		}
		params := DeploymentsQuery{Cluster: cluster, Namespace: namespace, Workload: workload}

		if err := extractDeploymentQueryParams(r, &params, nil); err != nil {
			RespondWithError(w, http.StatusBadRequest, err.Error())
			return
		}

		deployments, err := deploymentService.ListDeploymentsInRange(ctx, params.From, params.To)
		log.Tracef("deployments %+v\n", deployments)
		if err != nil {
			RespondWithError(w, http.StatusServiceUnavailable, err.Error())
			return
		}

		response := &DeploymentResponse{
			Deployments: deployments,
			Total:       len(deployments),
		}
		log.Tracef("response %+v\n", response)
		RespondWithJSON(w, http.StatusOK, response)
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

func extractRepoName(workload string) string {
	regexStr := "-v\\d.*"
	r, err := regexp.Compile(regexStr)
	if err != nil {
		log.Error(err)
		return ""
	}
	match, _ := regexp.MatchString(regexStr, workload)
	repoName := workload
	if match {
		repoName = r.ReplaceAllString(workload, "")
	}
	return repoName
}
