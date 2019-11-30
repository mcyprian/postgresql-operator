package e2e

import (
	"fmt"
	"testing"
	"time"

	goctx "context"

	postgresqlv1 "github.com/mcyprian/postgresql-operator/pkg/apis/postgresql/v1"
	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/operator-framework/operator-sdk/pkg/test/e2eutil"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func initializeTestEnvironment(t *testing.T, f *framework.Framework, ctx *framework.TestCtx) {
	if err := ctx.InitializeClusterResources(&framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval}); err != nil {
		t.Fatalf("Failed to initialize cluster resources: %v", err)
	}
	t.Log("Cluster resources initialized")
	namespace, err := ctx.GetNamespace()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Namespace found: %v", namespace)
	// wait for operator to be ready
	if err := e2eutil.WaitForDeployment(t, f.KubeClient, namespace, "postgresql-operator", 1, retryInterval, timeout); err != nil {
		t.Fatal(err)
	}

}

func newTestCluster(namespace string) *postgresqlv1.PostgreSQL {
	cpuValue, _ := resource.ParseQuantity("100m")
	memValue, _ := resource.ParseQuantity("250M")
	resources := corev1.ResourceRequirements{
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    cpuValue,
			corev1.ResourceMemory: memValue,
		},
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    cpuValue,
			corev1.ResourceMemory: memValue,
		},
	}

	primaryNode := postgresqlv1.PostgreSQLNode{
		Priority:  100,
		Resources: resources,
		Storage:   postgresqlv1.PostgreSQLStorageSpec{},
	}
	standbyNode := postgresqlv1.PostgreSQLNode{
		Priority:  20,
		Resources: resources,
		Storage:   postgresqlv1.PostgreSQLStorageSpec{},
	}

	return &postgresqlv1.PostgreSQL{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PostgreSQL",
			APIVersion: postgresqlv1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      postgreSQLCRName,
			Namespace: namespace,
		},
		Spec: postgresqlv1.PostgreSQLSpec{
			ManagementState: postgresqlv1.ManagementStateManaged,
			Nodes: map[string]postgresqlv1.PostgreSQLNode{
				"primary-node": primaryNode,
				"standby-node": standbyNode,
			},
		},
		Status: postgresqlv1.PostgreSQLStatus{
			Nodes: map[string]postgresqlv1.PostgreSQLNodeStatus{
				"primary-node": postgresqlv1.PostgreSQLNodeStatus{},
				"standby-node": postgresqlv1.PostgreSQLNodeStatus{},
			},
		},
	}
}

type getStatusFunc func(f *framework.Framework, namespace string) error

func retryExecution(t *testing.T, f *framework.Framework, namespace string, fce getStatusFunc, retries int, timeout time.Duration) error {
	attempt := -1
	for {
		attempt++
		t.Log(fmt.Sprintf("getStatus execution, attempt number %v", attempt))
		if attempt > 0 {
			time.Sleep(timeout)
		}
		if err := fce(f, namespace); err != nil {
			if attempt >= retries {
				return err
			}
		} else {
			return nil
		}
	}
}

func getStatusDouble(f *framework.Framework, namespace string) error {
	exampleName := types.NamespacedName{Name: postgreSQLCRName, Namespace: namespace}
	current := &postgresqlv1.PostgreSQL{}
	if err := f.Client.Get(goctx.TODO(), exampleName, current); err != nil {
		return fmt.Errorf("Failed to get examplePostgreSQL: %v", err)
	}
	for name, status := range current.Status.Nodes {
		if name == "primary-node" {
			if status.Role != "primary" {
				return fmt.Errorf("Wrong node role or status, expected (%v, %v), got: (%v, %v)", "primary-node", "primary", name, status.Role)
			}
		} else {
			if status.Role != "standby" {
				return fmt.Errorf("Wrong node role or status, expected (%v, %v), got: (%v, %v)", name, "standby", name, status.Role)
			}
		}
	}
	return nil
}

func getStatusSingle(f *framework.Framework, namespace string) error {
	exampleName := types.NamespacedName{Name: postgreSQLCRName, Namespace: namespace}
	current := &postgresqlv1.PostgreSQL{}

	if err := f.Client.Get(goctx.TODO(), exampleName, current); err != nil {
		return fmt.Errorf("Failed to get examplePostgreSQL: %v", err)
	}
	for name, status := range current.Status.Nodes {
		if name == "primary-node" {
			if status.Role != "primary" {
				return fmt.Errorf("Wrong node role or status, expected %v, got: %v", "primary", status.Role)
			}
		} else {
			return fmt.Errorf("Wrong node name %v, only primary-node should be present in the cluster", name)
		}
	}
	return nil
}

func getStatusFailover(f *framework.Framework, namespace string) error {
	exampleName := types.NamespacedName{Name: postgreSQLCRName, Namespace: namespace}
	current := &postgresqlv1.PostgreSQL{}

	if err := f.Client.Get(goctx.TODO(), exampleName, current); err != nil {
		return fmt.Errorf("Failed to get examplePostgreSQL: %v", err)
	}
	for name, status := range current.Status.Nodes {
		if name == "standby-node" {
			if status.Role != "primary" {
				return fmt.Errorf("Wrong node role or status, expected %v, got: %v", "primary", status.Role)
			}
		} else {
			return fmt.Errorf("Wrong node name %v, only standby-node should be present in the cluster", name)
		}
	}
	return nil
}
