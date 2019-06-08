package k8shandler

import (
	"context"
	"fmt"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func newHeadlessService(request *PostgreSQLRequest) *corev1.Service {
	service := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: appsv1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "postgresql",
			Namespace: request.cluster.Namespace,
			Labels:    NewLabels("postgresql", request.cluster.Name),
		},
		Spec: corev1.ServiceSpec{
			ClusterIP: "None",
			Selector:  selectorForPg("postgresql"),
			Ports: []corev1.ServicePort{
				corev1.ServicePort{
					Port:     postgresqlPort,
					Protocol: "TCP",
				},
			},
		},
	}
	controllerutil.SetControllerReference(request.cluster, service, request.scheme)
	return service
}

// CreateOrUpdateService creates a new Service if doesn't exists and ensures all its
// attributes has desired values
func CreateOrUpdateService(request *PostgreSQLRequest) error {
	service := newHeadlessService(request)

	err := request.client.Create(context.TODO(), service)
	if err != nil {
		if !errors.IsAlreadyExists(err) {
			return fmt.Errorf("Failure constructing %v service: %v", service.Name, err)
		}

		current := service.DeepCopy()
		retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			if err = request.client.Get(context.TODO(), types.NamespacedName{Name: request.cluster.Name, Namespace: request.cluster.Namespace}, current); err != nil {
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
