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

func newClusterIPService(request *PostgreSQLRequest, name string, primary bool) *corev1.Service {
	var selectorLabels map[string]string
	if primary {
		selectorLabels = newLabels(request.cluster.Name, "", true)
	} else {
		selectorLabels = newLabels(request.cluster.Name, name, false)
	}
	service := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: appsv1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: request.cluster.Namespace,
			Labels:    newLabels(request.cluster.Name, name, false),
		},
		Spec: corev1.ServiceSpec{
			Selector: selectorLabels,
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
func CreateOrUpdateService(request *PostgreSQLRequest, name string, primary bool) error {
	service := newClusterIPService(request, name, primary)
	if err := request.client.Create(context.TODO(), service); err != nil {
		if !errors.IsAlreadyExists(err) {
			return fmt.Errorf("Failed to construct service %v: %v", service.Name, err)
		}
		current := service.DeepCopy()
		retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			if err = request.client.Get(context.TODO(), types.NamespacedName{Name: request.cluster.Name, Namespace: request.cluster.Namespace}, current); err != nil {
				if errors.IsNotFound(err) {
					// the object doesn't exist -- it was likely culled
					// recreate it on the next time through if necessary
					return nil
				}
				return fmt.Errorf("Failed to get service %v: %v", service.Name, err)
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
