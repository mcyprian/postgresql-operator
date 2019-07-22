package k8shandler

import (
	postgresqlv1 "github.com/mcyprian/postgresql-operator/pkg/apis/postgresql/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	client "sigs.k8s.io/controller-runtime/pkg/client"
)

type PostgreSQLRequest struct {
	client     client.Client
	cluster    *postgresqlv1.PostgreSQL
	scheme     *runtime.Scheme
	restConfig *rest.Config
	clientset  *kubernetes.Clientset
}

func NewPostgreSQLRequest(client client.Client, cluster *postgresqlv1.PostgreSQL, scheme *runtime.Scheme) *PostgreSQLRequest {
	var clientset *kubernetes.Clientset
	config, err := rest.InClusterConfig()
	if err != nil {
		log.Error(err, "Failed to create rest config")
	} else {
		clientset, err = kubernetes.NewForConfig(config)
		if err != nil {
			log.Error(err, "Failed to create clientset")
		}
	}
	return &PostgreSQLRequest{client: client, cluster: cluster, scheme: scheme, restConfig: config, clientset: clientset}
}

func Reconcile(request *PostgreSQLRequest) (bool, error) {
	reqLogger := log.WithValues("Request.Namespace", request.cluster.Namespace, "Request.Name", request.cluster.Name)
	reqLogger.Info("Reconciling PostgreSQL")

	reqLogger.Info("Running create or update for Secret")
	if err := CreateOrUpdateSecret(request); err != nil {
		reqLogger.Error(err, "Failed to create or update Secret")
		return true, err
	}

	reqLogger.Info("Running create or update for ConfigMap")
	if err := CreateOrUpdateConfigMap(request); err != nil {
		reqLogger.Error(err, "Failed to create or update ConfigMap")
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
