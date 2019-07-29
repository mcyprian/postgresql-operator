package k8shandler

import (
	postgresqlv1 "github.com/mcyprian/postgresql-operator/pkg/apis/postgresql/v1"
	"k8s.io/apimachinery/pkg/runtime"
	client "sigs.k8s.io/controller-runtime/pkg/client"
)

// PostgreSQLRequest encapsulates variables needed for request handling
type PostgreSQLRequest struct {
	client  client.Client
	cluster *postgresqlv1.PostgreSQL
	scheme  *runtime.Scheme
}

// NewPostgreSQLRequest constructs a PostgreSQLRequest
func NewPostgreSQLRequest(client client.Client, cluster *postgresqlv1.PostgreSQL, scheme *runtime.Scheme) *PostgreSQLRequest {
	return &PostgreSQLRequest{client: client, cluster: cluster, scheme: scheme}
}

// Reconcile creates or updates all the resources managed by the operator
func Reconcile(request *PostgreSQLRequest) (bool, error) {
	reqLogger := log.WithValues("Request.Namespace", request.cluster.Namespace, "Request.Name", request.cluster.Name)
	reqLogger.Info("Reconciling PostgreSQL")

	reqLogger.Info("Running create or update for Secret")
	if err := CreateOrUpdateSecret(request); err != nil {
		reqLogger.Error(err, "Failed to create or update Secret")
		return true, err
	}

	reqLogger.Info("Running create or update for primary Service")
	err := CreateOrUpdateService(request, "postgresql-primary", true)
	if err != nil {
		reqLogger.Error(err, "Failed to create or update Service")
		return true, err
	}

	reqLogger.Info("Running create or update for Cluster")
	requeue, err := CreateOrUpdateCluster(request)
	if err != nil {
		reqLogger.Error(err, "Failed to create or update Cluster")
		return true, err
	} else if requeue {
		reqLogger.Info("Request requeued after create or update of Cluster")
		return true, nil
	}
	return false, nil
}
