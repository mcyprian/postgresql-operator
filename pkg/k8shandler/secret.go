package k8shandler

import (
	"context"
	"fmt"
	api "github.com/mcyprian/postgresql-operator/pkg/apis/postgresql/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func newSecret(p *api.PostgreSQL, secretName string) *corev1.Secret {
	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: p.Namespace,
		},
		StringData: map[string]string{
			"database-user":     "${POSTGRESQL_USER}",
			"database-password": "${POSTGRESQL_PASSWORD}",
			"database-name":     "${POSTGRESQL_DATABASE}",
		},
	}
}

// CreateOrUpdateSecret creates a new Secret if doesn't exists and ensures all its
// attributes has desired values
func CreateOrUpdateSecret(p *api.PostgreSQL, client client.Client) error {
	secret := newSecret(p, p.Name)

	err := client.Create(context.TODO(), secret)
	if err != nil {
		if !errors.IsAlreadyExists(err) {
			return fmt.Errorf("Failure constructing %v secret: %v", secret.Name, err)
		}

		current := secret.DeepCopy()
		retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			if err = client.Get(context.TODO(), types.NamespacedName{Name: p.Name, Namespace: p.Namespace}, current); err != nil {
				if errors.IsNotFound(err) {
					// the object doesn't exist -- it was likely culled
					// recreate it on the next time through if necessary
					return nil
				}
				return fmt.Errorf("Failed to get %v secret: %v", secret.Name, err)
			}

			current.StringData = secret.StringData
			if err = client.Update(context.TODO(), current); err != nil {
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
