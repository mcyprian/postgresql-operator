package k8shandler

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
)

// newContainer returns a container object for postgresql pod
func newContainer(name, secretName, image string, resourceRequirements corev1.ResourceRequirements, nodeID int, operation string) corev1.Container {
	env := newPgEnvironment()
	return corev1.Container{
		Image:   image,
		Name:    name,
		Command: []string{defaultCntCommand},
		Ports: []corev1.ContainerPort{{
			ContainerPort: postgresqlPort,
			Name:          "postgresql",
		}},
		ReadinessProbe: &corev1.Probe{
			TimeoutSeconds:      10,
			InitialDelaySeconds: 10,
			PeriodSeconds:       5,
			Handler: corev1.Handler{
				Exec: &corev1.ExecAction{
					Command: []string{defaultHealthCheckCommand},
				},
			},
		},
		LivenessProbe: &corev1.Probe{
			TimeoutSeconds:      10,
			InitialDelaySeconds: 60,
			PeriodSeconds:       10,
			Handler: corev1.Handler{
				Exec: &corev1.ExecAction{
					Command: []string{defaultHealthCheckCommand},
				},
			},
		},
		Env: []corev1.EnvVar{
			corev1.EnvVar{
				Name:  "POSTGRESQL_USER",
				Value: env.user,
			},
			corev1.EnvVar{
				Name: "POSTGRESQL_PASSWORD",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{Name: secretName},
						Key:                  "database-password",
					},
				},
			},
			corev1.EnvVar{
				Name:  "POSTGRESQL_DATABASE",
				Value: env.database,
			},
			corev1.EnvVar{
				Name:  "PGPASSFILE",
				Value: pgpassFilePath,
			},
			corev1.EnvVar{
				Name:  "STARTUP_OPERATION",
				Value: operation,
			},
			corev1.EnvVar{
				Name:  "ENABLE_REPMGR",
				Value: "true",
			},
			corev1.EnvVar{
				Name: "REPMGR_PASSWORD",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{Name: secretName},
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
				Value: fmt.Sprintf("%v", nodeID),
			},
		},
		Resources: resourceRequirements,
		VolumeMounts: []corev1.VolumeMount{
			corev1.VolumeMount{
				Name:      name,
				MountPath: pgDataPath,
			},
		},
	}
}
