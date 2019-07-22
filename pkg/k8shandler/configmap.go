package k8shandler

import (
	"context"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func newConfigMap(request *PostgreSQLRequest, name string) *corev1.ConfigMap {
	configMap := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: request.cluster.Namespace,
		},
		Data: map[string]string{
			"repmgr.conf": `
node_id='1'    
node_name='node-one'    
conninfo='host='node-one' user=repmgr dbname=repmgr connect_timeout=2'    
data_directory='${PGDATA}'    
use_replication_slots = 1    
failover = automatic    
promote_command='repmgr -f /app/repmgr.conf standby promote'    
follow_command='repmgr -f /app/repmgr.conf standby follow --upstream-node-id=%n'
            `,
		},
	}
	// Set PostgreSQL instance as the owner and controller
	controllerutil.SetControllerReference(request.cluster, configMap, request.scheme)
	return configMap
}

// CreateOrUpdateSecret creates a new Secret if doesn't exists and ensures all its
// attributes has desired values
func CreateOrUpdateConfigMap(request *PostgreSQLRequest) error {
	configMap := newConfigMap(request, "repmgr-conf")

	err := request.client.Create(context.TODO(), configMap)
	if err != nil {
		if !errors.IsAlreadyExists(err) {
			return fmt.Errorf("Failure constructing %v configMap: %v", configMap.Name, err)
		}

		current := configMap.DeepCopy()
		retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			if err = request.client.Get(context.TODO(), types.NamespacedName{Name: request.cluster.Name, Namespace: request.cluster.Namespace}, current); err != nil {
				if errors.IsNotFound(err) {
					// the object doesn't exist -- it was likely culled
					// recreate it on the next time through if necessary
					return nil
				}
				return fmt.Errorf("Failed to get %v configMap: %v", configMap.Name, err)
			}

			current.Data = configMap.Data
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
