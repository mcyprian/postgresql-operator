package k8shandler

import (
	"context"
	"fmt"

	postgresqlv1 "github.com/mcyprian/postgresql-operator/pkg/apis/postgresql/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type deploymentNode struct {
	self *appsv1.Deployment
	svc  *corev1.Service
}

func newDeploymentNode(request *PostgreSQLRequest, name string, specNode *postgresqlv1.PostgreSQLNode) *deploymentNode {
	return &deploymentNode{
		self: newDeployment(request, name, specNode),
		svc:  newClusterIPService(request, name),
	}
}

func (node *deploymentNode) name() string {
	return node.self.ObjectMeta.Name
}

func (node *deploymentNode) create(request *PostgreSQLRequest) error {
	err := request.client.Create(context.TODO(), node.self)
	if err != nil {
		if !errors.IsAlreadyExists(err) {
			return fmt.Errorf("Failed to create node resource %v", err)
		}
	}
	err = CreateOrUpdateService(request, node.svc)
	if err != nil {
		return fmt.Errorf("Failed to create service resource %v", err)
	}
	return nil
}

func (node *deploymentNode) update(request *PostgreSQLRequest, specNode *postgresqlv1.PostgreSQLNode) (bool, error) {
	// TODO update node to reflect spec
	//err := CreateOrUpdateService(request, node.svc)
	//if err != nil {
	//	return fmt.Errorf("Failed to create service resource %v", err)
	//}
	return false, nil
}

func (node *deploymentNode) delete(request *PostgreSQLRequest) error {
	err := request.client.Delete(context.TODO(), node.self)
	if err != nil {
		return fmt.Errorf("Failed to delete node resource %v", err)
	}
	err = request.client.Delete(context.TODO(), node.svc)
	if err != nil {
		return fmt.Errorf("Failed to delete service resource %v", err)
	}
	return nil
}

func (node *deploymentNode) status() postgresqlv1.PostgreSQLNodeStatus {
	// TODO Return node role and status
	return postgresqlv1.PostgreSQLNodeStatus{
		DeploymentName: node.self.ObjectMeta.Name,
		ServiceName:    node.svc.ObjectMeta.Name,
	}
}

//newDeployment returns a postgresql node Deployment object
func newDeployment(request *PostgreSQLRequest, name string, node *postgresqlv1.PostgreSQLNode) *appsv1.Deployment {
	var single int32 = 1
	labels := newLabels(request.cluster.Name, name)
	resourceRequirements := newResourceRequirements(node.Resources)
	deployment := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: appsv1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: request.cluster.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &single,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Strategy: appsv1.DeploymentStrategy{
				Type: "Recreate",
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Hostname:   name,
					Containers: []corev1.Container{newPostgreSQLContainer(name, resourceRequirements)},
				},
			},
		},
	}
	// Set PostgreSQL instance as the owner and controller
	controllerutil.SetControllerReference(request.cluster, deployment, request.scheme)
	return deployment
}
