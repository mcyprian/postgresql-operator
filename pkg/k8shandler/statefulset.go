package k8shandler

import (
	postgresqlv1 "github.com/mcyprian/postgresql-operator/pkg/apis/postgresql/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// CreateStatefulSet returns a postgresql StatefulSet object
func CreateStatefulSet(p *postgresqlv1.PostgreSQL, ctlscheme *runtime.Scheme) *appsv1.StatefulSet {
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
			Replicas: &replicas,
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
		},
	}
	// Set PostgreSQL instance as the owner and controller
	if ctlscheme != nil {
		controllerutil.SetControllerReference(p, set, ctlscheme)
	}
	return set
}

func newPostgreSQLContainer() corev1.Container {
	return corev1.Container{
		Image:   "mcyprian/postgresql-10-fedora29",
		Name:    "postgresql",
		Command: []string{"run-postgresql"},
		Ports: []corev1.ContainerPort{{
			ContainerPort: 5432,
			Name:          "postgresql",
		}},
		ReadinessProbe: &corev1.Probe{
			TimeoutSeconds:      30,
			InitialDelaySeconds: 10,
			PeriodSeconds:       5,
			Handler: corev1.Handler{
				Exec: &corev1.ExecAction{
					Command: []string{"/usr/libexec/check-container"},
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
				Name:  "STANDBY",
				Value: "false",
			},
			corev1.EnvVar{
				Name:  "ENABLE_REPLICATION",
				Value: "true",
			},
		},
	}
}
