package k8shandler

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func newSecret(request *PostgreSQLRequest, name string, passwords *pgPasswords) *corev1.Secret {
	secret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: request.cluster.Namespace,
		},
		StringData: map[string]string{
			"database-password": passwords.database,
			"repmgr-password":   passwords.repmgr,
		},
	}
	// Set PostgreSQL instance as the owner and controller
	controllerutil.SetControllerReference(request.cluster, secret, request.scheme)
	return secret
}

// CreateOrUpdateSecret creates a new Secret if doesn't exists and ensures all its
// attributes has desired values
func (request *PostgreSQLRequest) CreateOrUpdateSecret() error {
	_, err := extractSecret(request.cluster.Name, request.cluster.Namespace, request.client)
	if errors.IsNotFound(err) {
		log.Info(fmt.Sprintf("Generating secret for cluster %v", request.cluster.Name))
		passwords, err := newPgPasswords()
		if err != nil {
			log.Error(err, "Failed to generate passwords")
			return err
		}
		secret := newSecret(request, request.cluster.Name, passwords)
		if err := request.client.Create(context.TODO(), secret); err != nil {
			if !errors.IsAlreadyExists(err) {
				return fmt.Errorf("Failure constructing %v secret: %v", secret.Name, err)
			}

		}
	}
	return nil
}

func extractSecret(secretName, namespace string, client client.Client) (map[string][]byte, error) {
	secret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: corev1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: namespace,
		},
	}

	if err := client.Get(context.TODO(), types.NamespacedName{Name: secret.Name, Namespace: secret.Namespace}, secret); err != nil {
		if errors.IsNotFound(err) {
			log.Error(err, fmt.Sprintf("Failed to find secret %v", secret.Name))
		} else {
			log.Error(err, fmt.Sprintf("Failed to read secret %v", secret.Name))
		}
		return nil, err
	}
	return secret.Data, nil
}
