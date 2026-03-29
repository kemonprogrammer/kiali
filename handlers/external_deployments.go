package handlers

import (
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

type DeploymentResponse struct {
	Deployments []*model.Deployment `json:"deployments"`
	Total       int                 `json:"total"`
}

// WorkloadExternalDeployments is the API handler to fetch GitHub deployments, related to a single workload
func WorkloadExternalDeployments(
	conf *config.Config,
	cache cache.KialiCache,
	clientFactory kubernetes.ClientFactory,
	discovery istio.MeshDiscovery,
	deploymentClient external_deployments.DeploymentClient,
) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		vars := mux.Vars(r)
		namespace := vars["namespace"]
		workload := vars["workload"]

		if len(workload) == 0 {
			RespondWithError(w, http.StatusBadRequest, "No workload provided!")
			return
		}

		repo := extractRepoName(workload)
		ds := external_deployments.NewDeploymentService(conf, deploymentClient)
		if err := ds.SetRepo(ctx, repo); err != nil {
			RespondWithError(w, http.StatusServiceUnavailable, fmt.Sprintf("no repository found for workload %s %s", workload, err))
			return
		}

		cluster := clusterNameFromQuery(conf, r.URL.Query())
		if _, err := checkNamespaceAccess(w, r, conf, cache, discovery, clientFactory, namespace, cluster); err != nil {
			RespondWithError(w, http.StatusServiceUnavailable, err.Error())
			return
		}
		params := models.DeploymentsQuery{Cluster: cluster, Namespace: namespace, Repository: repo}

		if err := extractDeploymentQueryParams(r, &params, nil); err != nil {
			RespondWithError(w, http.StatusBadRequest, err.Error())
			return
		}

		deployments, err := ds.ListDeploymentsInRange(ctx, params)
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

// ServiceExternalDeployments is the API handler to fetch GitHub deployments, related to a single service
func ServiceExternalDeployments(
	conf *config.Config,
	cache cache.KialiCache,
	clientFactory kubernetes.ClientFactory,
	discovery istio.MeshDiscovery,
	deploymentClient external_deployments.DeploymentClient,
) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		vars := mux.Vars(r)
		namespace := vars["namespace"]
		service := vars["service"]

		if len(service) == 0 {
			RespondWithError(w, http.StatusBadRequest, "No service provided!")
			return
		}

		repo := extractRepoName(service)
		ds := external_deployments.NewDeploymentService(conf, deploymentClient)

		if err := ds.SetRepo(ctx, repo); err != nil {
			RespondWithError(w, http.StatusServiceUnavailable, fmt.Sprintf("no repository found for service %s %s", service, err))
			return
		}

		cluster := clusterNameFromQuery(conf, r.URL.Query())
		if _, err := checkNamespaceAccess(w, r, conf, cache, discovery, clientFactory, namespace, cluster); err != nil {
			RespondWithError(w, http.StatusServiceUnavailable, err.Error())
			return
		}
		params := models.DeploymentsQuery{Cluster: cluster, Namespace: namespace, Repository: repo}

		if err := extractDeploymentQueryParams(r, &params, nil); err != nil {
			RespondWithError(w, http.StatusBadRequest, err.Error())
			return
		}

		deployments, err := ds.ListDeploymentsInRange(ctx, params)
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

// AppExternalDeployments is the API handler to fetch GitHub deployments, related to a single app
func AppExternalDeployments(
	conf *config.Config,
	cache cache.KialiCache,
	clientFactory kubernetes.ClientFactory,
	discovery istio.MeshDiscovery,
	deploymentClient external_deployments.DeploymentClient,
) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		vars := mux.Vars(r)
		namespace := vars["namespace"]
		app := vars["app"]

		if len(app) == 0 {
			RespondWithError(w, http.StatusBadRequest, "No app provided!")
			return
		}

		repo := extractRepoName(app)
		ds := external_deployments.NewDeploymentService(conf, deploymentClient)
		if err := ds.SetRepo(ctx, repo); err != nil {
			RespondWithError(w, http.StatusServiceUnavailable, fmt.Sprintf("no repository found for app %s %s", app, err))
			return
		}

		cluster := clusterNameFromQuery(conf, r.URL.Query())
		if _, err := checkNamespaceAccess(w, r, conf, cache, discovery, clientFactory, namespace, cluster); err != nil {
			RespondWithError(w, http.StatusServiceUnavailable, err.Error())
			return
		}
		params := models.DeploymentsQuery{Cluster: cluster, Namespace: namespace, Repository: repo}

		if err := extractDeploymentQueryParams(r, &params, nil); err != nil {
			RespondWithError(w, http.StatusBadRequest, err.Error())
			return
		}

		deployments, err := ds.ListDeploymentsInRange(ctx, params)
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

func extractDeploymentQueryParams(r *http.Request, query *models.DeploymentsQuery, namespaceInfo *models.Namespace) error {
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
