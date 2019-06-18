package k8shandler

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	postgresqlv1 "github.com/mcyprian/postgresql-operator/pkg/apis/postgresql/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
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
	err := CreateOrUpdateSecret(request)
	if err != nil {
		reqLogger.Error(err, "Failed to create of update Secret")
		return true, err
	}

	//reqLogger.Info("Running create or update for StatefulSet")
	//requeue, err := CreateOrUpdateStatefulSet(request)
	//if err != nil {
	//	reqLogger.Error(err, "Failed to create of update StatefulSet")
	//	return true, err
	//} else if requeue {
	//	return true, nil
	//}
	reqLogger.Info("Running create or update for Cluster")
	requeue, err := CreateOrUpdateCluster(request)
	if err != nil {
		reqLogger.Error(err, "Failed to create of update Cluster")
		return true, err
	} else if requeue {
		return true, nil
	}

	//reqLogger.Info("Running create or update for Service")
	//err = CreateOrUpdateService(request)
	//if err != nil {
	//	reqLogger.Error(err, "Failed to create of update Service")
	//	return true, err
	//}

	// Define a new Pod object
	podList := &corev1.PodList{}
	labelSelector := labels.SelectorFromSet(newLabels(request.cluster.Name, ""))
	listOps := &client.ListOptions{Namespace: request.cluster.Namespace, LabelSelector: labelSelector}
	err = request.client.List(context.TODO(), listOps, podList)
	if err != nil {
		reqLogger.Error(err, "Failed to list pods", "PostgreSQL.Namespace", request.cluster.Namespace, "PostgreSQL.Name", request.cluster.Name)
		return true, err
	}
	podNames := getPodNames(podList.Items)

	// Update status.Nodes if needed
	if !reflect.DeepEqual(podNames, request.cluster.Status.Nodes) {
		request.cluster.Status.Nodes = podNames
		err := request.client.Status().Update(context.TODO(), request.cluster)
		if err != nil {
			reqLogger.Error(err, "Failed to update PostgreSQL status")
			return true, err
		}
	}
	// Register nodes which were not registered to repmgr cluster yet
	repmgrClusterUp := true
	if len(podList.Items) != len(request.cluster.Spec.Nodes) {
		repmgrClusterUp = false
	}
	for _, pod := range podList.Items {
		if !isReady(pod) {
			repmgrClusterUp = false
		} else {
			registered, _ := isRegistered(request, pod)
			if !registered {
				err = repmgrRegister(request, pod)
				if err != nil {
					reqLogger.Error(err, "Repmgr register failed")
					return true, err
				}
				repmgrClusterUp = false
			}
		}
	}
	if !repmgrClusterUp {
		return true, nil
	}
	return false, nil
}

// getPodNames returns the pod names of the array of pods passed in
func getPodNames(pods []corev1.Pod) []string {
	var podNames []string
	for _, pod := range pods {
		podNames = append(podNames, pod.Name)
	}
	return podNames
}

// isReady determines whether pod status is Ready
func isReady(pod corev1.Pod) bool {
	for _, cond := range pod.Status.Conditions {
		if cond.Type == "Ready" && cond.Status == "True" {
			return true
		}
	}
	return false
}

// isRegistered determines whether repmgr node was successfuly registered
func isRegistered(request *PostgreSQLRequest, pod corev1.Pod) (bool, error) {
	execCommand := []string{"shell-entrypoint", "repmgr", "node", "check"}
	stdout, stderr, err := ExecToPodThroughAPI(request.restConfig, request.clientset, execCommand, pod.Spec.Containers[0].Name, pod.Name, request.cluster.Namespace, nil)
	if err != nil {
		log.Info(fmt.Sprintf("Repmgr node check returned non-zero exit status, stdout: %v, stderr: %v", stdout, stderr))
		return false, err
	} else {
		log.Info("Repmgr node check executed", stdout, stderr)
		if strings.Contains(stdout, "OK") {
			return true, nil
		} else {
			return false, nil
		}
	}
}

func repmgrRegister(request *PostgreSQLRequest, pod corev1.Pod) error {
	execCommand := []string{"shell-entrypoint", "repmgr-register"}
	stdout, stderr, err := ExecToPodThroughAPI(request.restConfig, request.clientset, execCommand, pod.Spec.Containers[0].Name, pod.Name, request.cluster.Namespace, nil)
	if err != nil {
		log.Error(err, fmt.Sprintf("Repmgr register failed, stdout: %v, stderr: %v", stdout, stderr))
		return err
	} else {
		log.Info("Repmgr register executed", stdout, stderr)
	}
	return nil
}
