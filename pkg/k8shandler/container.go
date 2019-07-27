package k8shandler

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
)

// newContainer returns a container object for postgresql pod
func newContainer(name string, resourceRequirements corev1.ResourceRequirements, nodeId int, primary bool) corev1.Container {
	var command string = defaultCntCommand
	if primary {
		command = defaultCntCommandPrimary
	}
	return corev1.Container{
		Image:   defaultPgImage,
		Name:    name,
		Command: []string{command},
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
		Env: []corev1.EnvVar{
			corev1.EnvVar{
				Name:  "POSTGRESQL_USER",
				Value: defaultPgUser,
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
				Value: defaultPgDatabase,
			},
			corev1.EnvVar{
				Name:  "PGPASSFILE",
				Value: "/var/lib/pgsql/.pgpass",
			},
			corev1.EnvVar{
				Name:  "ENABLE_REPMGR",
				Value: "true",
			},
			corev1.EnvVar{
				Name: "REPMGR_PASSWORD",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{Name: "postgresql"},
						Key:                  "repmgr-password",
					},
				},
			},
			corev1.EnvVar{
				Name:  "NODE_NAME",
				Value: name,
			},
			corev1.EnvVar{
				Name:  "NODE_ID",
				Value: fmt.Sprintf("%v", nodeId),
			},
		},
		Resources: resourceRequirements,
		VolumeMounts: []corev1.VolumeMount{
			corev1.VolumeMount{
				Name:      name,
				MountPath: postgreSQLDataPath,
			},
		},
	}
}
