package k8shandler

import (
	"context"

	postgresqlv1 "github.com/mcyprian/postgresql-operator/pkg/apis/postgresql/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

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

// deleteNode removes node from the cluster
func deleteNode(request *PostgreSQLRequest, node *appsv1.Deployment) {
	request.client.Delete(context.TODO(), node)
}
