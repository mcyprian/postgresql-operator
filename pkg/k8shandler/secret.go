package k8shandler

import (
	"context"
	"fmt"
	"os"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func newSecret(request *PostgreSQLRequest, name string, repmgrPassword string, userPassword string) *corev1.Secret {
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
			"database-password": userPassword,
			"repmgr-password":   repmgrPassword,
		},
	}
	// Set PostgreSQL instance as the owner and controller
	controllerutil.SetControllerReference(request.cluster, secret, request.scheme)
	return secret
}

// CreateOrUpdateSecret creates a new Secret if doesn't exists and ensures all its
// attributes has desired values
func CreateOrUpdateSecret(request *PostgreSQLRequest) error {
	repmgrPassword, err := generatePassword()
	if err != nil {
		return fmt.Errorf("Failed to generate password for repmgr user: %v", err)
	}
	userPassword := os.Getenv("POSTGRESQL_PASSWORD")
	if userPassword == "" {
		userPassword, err = generatePassword()
		if err != nil {
			return fmt.Errorf("Failed to generate password for database user: %v", err)
		}
	}
	secret := newSecret(request, "postgresql", repmgrPassword, userPassword)

	if err = request.client.Create(context.TODO(), secret); err != nil {
		if !errors.IsAlreadyExists(err) {
			return fmt.Errorf("Failure constructing %v secret: %v", secret.Name, err)
		}

		current := secret.DeepCopy()
		retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			if err = request.client.Get(context.TODO(), types.NamespacedName{Name: request.cluster.Name, Namespace: request.cluster.Namespace}, current); err != nil {
				if errors.IsNotFound(err) {
					// the object doesn't exist -- it was likely culled
					// recreate it on the next time through if necessary
					return nil
				}
				return fmt.Errorf("Failed to get %v secret: %v", secret.Name, err)
			}

			current.StringData = secret.StringData
			if err = request.client.Update(context.TODO(), current); err != nil {
				return err
			}
			return nil
		})
		if retryErr != nil {
			return retryErr
		}
	}
	return nil
}
