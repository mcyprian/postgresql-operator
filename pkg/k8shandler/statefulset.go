package k8shandler

import (
	"context"
	"fmt"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// newStatefulSet returns a postgresql StatefulSet object
func newStatefulSet(request *PostgreSQLRequest) *appsv1.StatefulSet {
	labels := NewLabels("postgresql", request.cluster.Name)
	replicas := int32(len(request.cluster.Spec.Nodes))

	set := &appsv1.StatefulSet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "StatefulSet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      request.cluster.Name,
			Namespace: request.cluster.Namespace,
		},
		Spec: appsv1.StatefulSetSpec{
			ServiceName: "postgresql",
			Replicas:    &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{newPostgreSQLContainer(request.cluster.Name)},
				},
			},
			UpdateStrategy: appsv1.StatefulSetUpdateStrategy{
				Type:          appsv1.RollingUpdateStatefulSetStrategyType,
				RollingUpdate: &appsv1.RollingUpdateStatefulSetStrategy{},
			},
		},
	}
	// Set PostgreSQL instance as the owner and controller
	controllerutil.SetControllerReference(request.cluster, set, request.scheme)
	return set
}

// CreateOrUpdateStatus creates a new StatefulSet if doesn't exists and ensures the desired number
// of replicas is running
func CreateOrUpdateStatefulSet(request *PostgreSQLRequest) (bool, error) {
	set := &appsv1.StatefulSet{}
	err := request.client.Get(context.TODO(), types.NamespacedName{Name: request.cluster.Name, Namespace: request.cluster.Namespace}, set)
	if err != nil && errors.IsNotFound(err) {
		// Define a new statefulset
		current := newStatefulSet(request)
		err = request.client.Create(context.TODO(), current)
		if err != nil {
			return true, fmt.Errorf("Failed to create new StatefulSet", "StatefulSet.Namespace", current.Namespace, "StatefulSet.Name, %v", current.Name, err)
		}
		// StatefulSet created successfully - return and requeue
		return true, nil
	} else if err != nil {
		return true, fmt.Errorf("Failed to get %v StatefulSet: %v", set.Name, err)
	}

	// Ensure the set size is the same as the spec
	size := int32(len(request.cluster.Spec.Nodes))
	if *set.Spec.Replicas != size {
		set.Spec.Replicas = &size
		err = request.client.Update(context.TODO(), set)
		if err != nil {
			return true, fmt.Errorf("Failed to update StatefulSet", "StatefulSet.Namespace", set.Namespace, "StatefulSet.Name, %v", set.Name, err)
		}
		// Spec updated - return and requeue
		return true, nil
	}
	return false, nil
}
