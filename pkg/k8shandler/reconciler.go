package k8shandler

import (
	postgresqlv1 "github.com/mcyprian/postgresql-operator/pkg/apis/postgresql/v1"
	"github.com/sirupsen/logrus"
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
func (request *PostgreSQLRequest) Reconcile() (bool, error) {
	var err error
	logrus.Info("Reconciling PostgreSQL")

	logrus.Info("Running create or update for secret")
	if err := request.CreateOrUpdateSecret(); err != nil {
		logrus.Errorf("Failed to create or update secret: %v", err)
		return true, err
	}

	logrus.Info("Running create or update for read-only service")
	if err := request.CreateOrUpdateService("postgresql-ro", ""); err != nil {
		logrus.Errorf("Failed to create or update read-only secret: %v", err)
		return true, err
	}

	logrus.Info("Running create or update for cluster")
	requeue, err := request.CreateOrUpdateCluster()
	if err != nil {
		logrus.Errorf("Failed to create or update cluster: %v", err)
		return true, err
	} else if requeue {
		logrus.Info("Request requeued after create or update of cluster")
		return true, nil
	}
	return false, nil
}
