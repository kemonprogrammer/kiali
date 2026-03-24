package models

import "time"

type DeploymentsQuery struct {
	From, To                     time.Time
	Cluster, Namespace, Workload string
}
