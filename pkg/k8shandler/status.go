package k8shandler

import (
	"context"
	"fmt"
	"reflect"

	postgresqlv1 "github.com/mcyprian/postgresql-operator/pkg/apis/postgresql/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
)

// UpdateClusterStatus compares current and new status and perfroms the update if needed.
func UpdateClusterStatus(request *PostgreSQLRequest, clusterStatus *postgresqlv1.PostgreSQLStatus) error {
	if !reflect.DeepEqual(request.cluster.Status, clusterStatus) {
		nretries := -1
		retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			nretries++
			if err := request.client.Get(context.TODO(), types.NamespacedName{Name: request.cluster.Name, Namespace: request.cluster.Namespace}, request.cluster); err != nil {
				return fmt.Errorf("Couldn't get cluster: %v", err)
			}
			request.cluster.Status.Nodes = clusterStatus.Nodes

			if err := request.client.Update(context.TODO(), request.cluster); err != nil {
				return fmt.Errorf("Failed to update cluster status: %v", err)
			}
			return nil
		})
		if retryErr != nil {
			return fmt.Errorf("Couldn't update cluster status after %v retries: %v", nretries, retryErr)
		}
	}
	return nil
}
