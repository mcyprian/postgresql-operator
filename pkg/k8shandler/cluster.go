package k8shandler

import (
	"context"
	"fmt"
	postgresqlv1 "github.com/mcyprian/postgresql-operator/pkg/apis/postgresql/v1"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
)

// CreateOrUpdateCluster iterates over all desired nodes and performs create or update action on them
func CreateOrUpdateCluster(request *PostgreSQLRequest) (bool, error) {
	var requeue bool
	var err error
	for index := range request.cluster.Spec.Nodes {
		requeue, err = CreateOrUpdateNode(request, &request.cluster.Spec.Nodes[index])
		if err != nil {
			return true, err
		}
	}
	return requeue, nil
}

// CreateOrUpdateNode creates a new Deployment with specified node.GenUUID if doesn't exists and
// updates attributes not matching current spec
func CreateOrUpdateNode(request *PostgreSQLRequest, node *postgresqlv1.PostgreSQLNode) (bool, error) {
	deployment := &appsv1.Deployment{}
	err := request.client.Get(context.TODO(), types.NamespacedName{Name: *node.GenUUID, Namespace: request.cluster.Namespace}, deployment)
	if err != nil && errors.IsNotFound(err) {
		// Define a new deployemnt
		current := newDeployment(request, *node.GenUUID)
		err = request.client.Create(context.TODO(), current)
		if err != nil {
			return true, fmt.Errorf("Failed to create new Deployment %v, namespace %v: %v", *node.GenUUID, request.cluster.Namespace, err)
		}
		// Deployment created successfully - return and requeue
		return true, nil
	} else if err != nil {
		return true, fmt.Errorf("Failed to get Deployment %v: %v", node.GenUUID, err)
	}
	// TODO Check and update all the attributes image, resources, storage...
	// Spec updated - return and requeue (return true, nil)
	return false, nil
}
