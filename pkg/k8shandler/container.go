package k8shandler

import (
	corev1 "k8s.io/api/core/v1"
)

// newDeployment returns a container object for postgresql pod
func newPostgreSQLContainer(name string, resourceRequirements corev1.ResourceRequirements) corev1.Container {
	return corev1.Container{
		Image:   defaultPgImage,
		Name:    name,
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
				Name: "POSTGRESQL_PASSWORD",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{Name: "postgresql"},
						Key:                  "database-password",
					},
				},
			},
			corev1.EnvVar{
				Name:  "POSTGRESQL_DATABASE",
				Value: "db",
			},
			corev1.EnvVar{
				Name:  "POSTGRESQL_MASTER_SERVICE_NAME",
				Value: "postgresql-node-0",
			},
			corev1.EnvVar{
				Name:  "NODENAME",
				Value: name,
			},
		},
		Resources: resourceRequirements,
	}
}
