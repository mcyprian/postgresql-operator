package k8shandler

import (
	"context"
	"fmt"
	api "github.com/mcyprian/postgresql-operator/pkg/apis/postgresql/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func newHeadlessService(p *api.PostgreSQL) *corev1.Service {
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: appsv1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "postgresql",
			Namespace: p.Namespace,
			Labels:    NewLabels("postgresql", p.Name),
		},
		Spec: corev1.ServiceSpec{
			Selector: selectorForPg("postgresql"),
			Ports: []corev1.ServicePort{
				corev1.ServicePort{
					Port:     postgresqlPort,
					Protocol: "TCP",
				},
			},
		},
	}
}

// CreateOrUpdateService creates a new Service if doesn't exists and ensures all its
// attributes has desired values
func CreateOrUpdateService(p *api.PostgreSQL, client client.Client) error {
	service := newHeadlessService(p)

	err := client.Create(context.TODO(), service)
	if err != nil {
		if !errors.IsAlreadyExists(err) {
			return fmt.Errorf("Failure constructing %v service: %v", service.Name, err)
		}

		current := service.DeepCopy()
		retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			if err = client.Get(context.TODO(), types.NamespacedName{Name: p.Name, Namespace: p.Namespace}, current); err != nil {
				if errors.IsNotFound(err) {
					// the object doesn't exist -- it was likely culled
					// recreate it on the next time through if necessary
					return nil
				}
				return fmt.Errorf("Failed to get %v service: %v", service.Name, err)
			}

			current.Spec.Ports = service.Spec.Ports
			current.Spec.Selector = service.Spec.Selector
			current.Spec.PublishNotReadyAddresses = service.Spec.PublishNotReadyAddresses
			current.Labels = service.Labels
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
