package k8shandler

import (
	"context"
	"fmt"
	postgresqlv1 "github.com/mcyprian/postgresql-operator/pkg/apis/postgresql/v1"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	client "sigs.k8s.io/controller-runtime/pkg/client"
)

// CreateOrUpdateCluster iterates over all desired nodes and performs create or update action on
// related deployments and services
func CreateOrUpdateCluster(request *PostgreSQLRequest) (bool, error) {
	var requeue bool
	var err error

	reqLogger := log.WithValues("Request.Namespace", request.cluster.Namespace, "Request.Name", request.cluster.Name)
	// Run CreateOrUpdate for each node from the spec
	for name, node := range request.cluster.Spec.Nodes {
		requeue, err = CreateOrUpdateNode(request, name, &node)
		if err != nil {
			return true, err
		}
		err = CreateOrUpdateService(request, newClusterIPService(request, name))
		if err != nil {
			return true, err
		}
	}
	// Delete all extra nodes
	deploymentList := &appsv1.DeploymentList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: appsv1.SchemeGroupVersion.String(),
		},
	}
	labelSelector := labels.SelectorFromSet(newLabels(request.cluster.Name, ""))
	err = request.client.List(
		context.TODO(),
		&client.ListOptions{Namespace: request.cluster.Namespace, LabelSelector: labelSelector},
		deploymentList,
	)
	if err != nil {
		reqLogger.Error(err, "Failed to list deployments")
		return true, err
	}
	reqLogger.Info(fmt.Sprintf("List of deployments %v", deploymentList.Items))
	for _, deployment := range deploymentList.Items {
		_, ok := request.cluster.Spec.Nodes[deployment.ObjectMeta.Name]
		if !ok {
			reqLogger.Info("Deleting node", deployment)
			deleteNode(request, &deployment)
		}
	}
	return requeue, nil
}

// CreateOrUpdateNode creates a new Deployment with specified name if doesn't exists and
// updates attributes not matching current spec
func CreateOrUpdateNode(request *PostgreSQLRequest, name string, node *postgresqlv1.PostgreSQLNode) (bool, error) {
	deployment := &appsv1.Deployment{}
	err := request.client.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: request.cluster.Namespace}, deployment)
	if err != nil && errors.IsNotFound(err) {
		// Define a new deployemnt
		current := newDeployment(request, name, node)
		err = request.client.Create(context.TODO(), current)
		if err != nil {
			return true, fmt.Errorf("Failed to create new Deployment %v, namespace %v: %v", name, request.cluster.Namespace, err)
		}
		// Deployment created successfully - return and requeue
		return true, nil
	} else if err != nil {
		return true, fmt.Errorf("Failed to get Deployment %v: %v", name, err)
	}
	// TODO Check and update all the attributes image, resources, storage...
	// Spec updated - return and requeue (return true, nil)
	return false, nil
}
