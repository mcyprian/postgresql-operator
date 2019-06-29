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
	self        *appsv1.Deployment
	svc         *corev1.Service
	db          *database
	initPrimary bool
}

func newDeploymentNode(request *PostgreSQLRequest, name string, specNode *postgresqlv1.PostgreSQLNode, primary bool) *deploymentNode {
	return &deploymentNode{
		self:        newDeployment(request, name, specNode, primary),
		svc:         newClusterIPService(request, name, primary),
		db:          newRepmgrDatabase(name),
		initPrimary: primary,
	}
}

func (node *deploymentNode) name() string {
	return node.self.ObjectMeta.Name
}

func (node *deploymentNode) create(request *PostgreSQLRequest) error {
	if err := request.client.Create(context.TODO(), node.self); err != nil {
		if !errors.IsAlreadyExists(err) {
			return fmt.Errorf("Failed to create node resource %v", err)
		}
	}
	if err := CreateOrUpdateService(request, node.svc.ObjectMeta.Name, node.initPrimary); err != nil {
		return fmt.Errorf("Failed to create service resource %v", err)
	}
	if err := node.db.initialize(); err != nil {
		return fmt.Errorf("Failed to initialize repmgr database connection %v", err)
	}

	return nil
}

func (node *deploymentNode) update(request *PostgreSQLRequest, specNode *postgresqlv1.PostgreSQLNode) (bool, error) {
	// TODO update node to reflect spec
	//if err := CreateOrUpdateService(request, node.svc); err != nil {
	//	return fmt.Errorf("Failed to create service resource %v", err)
	//}
	return false, nil
}

func (node *deploymentNode) delete(request *PostgreSQLRequest) error {
	if err := request.client.Delete(context.TODO(), node.self); err != nil {
		return fmt.Errorf("Failed to delete node resource %v", err)
	}
	if err := request.client.Delete(context.TODO(), node.svc); err != nil {
		return fmt.Errorf("Failed to delete service resource %v", err)
	}
	node.db.engine.Close()
	return nil
}

func (node *deploymentNode) status() postgresqlv1.PostgreSQLNodeStatus {
	// TODO Return node role and status
	version, err := node.db.version()
	if err != nil {
		log.Error(err, "Failed to execute SQL query")
	}

	return postgresqlv1.PostgreSQLNodeStatus{
		DeploymentName: node.self.ObjectMeta.Name,
		ServiceName:    node.svc.ObjectMeta.Name,
		PgVersion:      version,
	}
}

//newDeployment returns a postgresql node Deployment object
func newDeployment(request *PostgreSQLRequest, name string, node *postgresqlv1.PostgreSQLNode, primary bool) *appsv1.Deployment {
	var single int32 = 1
	labels := newLabels(request.cluster.Name, name, primary)
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
