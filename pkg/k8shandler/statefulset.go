package k8shandler

import (
	"context"
	"fmt"
	api "github.com/mcyprian/postgresql-operator/pkg/apis/postgresql/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// newStatefulSet returns a postgresql StatefulSet object
func newStatefulSet(p *api.PostgreSQL, scheme *runtime.Scheme) *appsv1.StatefulSet {
	labels := NewLabels("postgresql", p.Name)
	replicas := p.Spec.Size

	set := &appsv1.StatefulSet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "StatefulSet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      p.Name,
			Namespace: p.Namespace,
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
					Containers: []corev1.Container{newPostgreSQLContainer()},
				},
			},
			UpdateStrategy: appsv1.StatefulSetUpdateStrategy{
				Type:          appsv1.RollingUpdateStatefulSetStrategyType,
				RollingUpdate: &appsv1.RollingUpdateStatefulSetStrategy{},
			},
		},
	}
	// Set PostgreSQL instance as the owner and controller
	if scheme != nil {
		controllerutil.SetControllerReference(p, set, scheme)
	}
	return set
}

func newPostgreSQLContainer() corev1.Container {
	return corev1.Container{
		Image:   defaultPgImage,
		Name:    "postgresql",
		Command: []string{defaultCntCommand},
		Ports: []corev1.ContainerPort{{
			ContainerPort: postgresqlPort,
			Name:          "postgresql",
		}},
		ReadinessProbe: &corev1.Probe{
			TimeoutSeconds:      30,
			InitialDelaySeconds: 10,
			PeriodSeconds:       5,
			Handler: corev1.Handler{
				Exec: &corev1.ExecAction{
					Command: []string{defaultHealthCheckCommand},
				},
			},
		},
		// TODO: rewrite to k8s secrets
		Env: []corev1.EnvVar{
			corev1.EnvVar{
				Name:  "POSTGRESQL_USER",
				Value: "user",
			},
			corev1.EnvVar{
				Name:  "POSTGRESQL_PASSWORD",
				Value: "secretpassword",
			},
			corev1.EnvVar{
				Name:  "POSTGRESQL_DATABASE",
				Value: "db",
			},
			corev1.EnvVar{
				Name:  "POSTGRESQL_MASTER_SERVICE_NAME",
				Value: "postgresql-node-0.postgresql",
			},
		},
	}
}

// CreateOrUpdateStatus creates a new StatefulSet if doesn't exists and ensures the desired number
// of replicas is running
func CreateOrUpdateStatefulSet(p *api.PostgreSQL, client client.Client, scheme *runtime.Scheme) (bool, error) {
	set := &appsv1.StatefulSet{}
	err := client.Get(context.TODO(), types.NamespacedName{Name: p.Name, Namespace: p.Namespace}, set)
	if err != nil && errors.IsNotFound(err) {
		// Define a new statefulset
		current := newStatefulSet(p, scheme)
		err = client.Create(context.TODO(), current)
		if err != nil {
			return true, fmt.Errorf("Failed to create new StatefulSet", "StatefulSet.Namespace", current.Namespace, "StatefulSet.Name, %v", current.Name, err)
		}
		// StatefulSet created successfully - return and requeue
		return true, nil
	} else if err != nil {
		return true, fmt.Errorf("Failed to get %v StatefulSet: %v", set.Name, err)
	}

	// Ensure the set size is the same as the spec
	size := p.Spec.Size
	if *set.Spec.Replicas != size {
		set.Spec.Replicas = &size
		err = client.Update(context.TODO(), set)
		if err != nil {
			return true, fmt.Errorf("Failed to update StatefulSet", "StatefulSet.Namespace", set.Namespace, "StatefulSet.Name, %v", set.Name, err)
		}

		// Spec updated - return and requeue
		return true, nil
	}
	return false, nil
}
