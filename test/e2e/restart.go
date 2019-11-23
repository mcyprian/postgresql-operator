package e2e

import (
	goctx "context"
	"fmt"
	"testing"
	"time"

	postgresqlv1 "github.com/mcyprian/postgresql-operator/pkg/apis/postgresql/v1"
	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/operator-framework/operator-sdk/pkg/test/e2eutil"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PostgreSQLClusterRestart test correct recovery of the cluster after slave and master node restart
func PostgreSQLClusterRestart(t *testing.T) {
	t.Parallel()
	ctx := framework.NewTestCtx(t)
	defer ctx.Cleanup()
	if err := ctx.InitializeClusterResources(&framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval}); err != nil {
		t.Fatalf("Failed to initialize cluster resources: %v", err)
	}
	t.Log("Cluster resources initialized")
	namespace, err := ctx.GetNamespace()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Namespace found: %v", namespace)
	// get global framework variables reference
	f := framework.Global
	// wait for operator to be ready
	if err := e2eutil.WaitForDeployment(t, f.KubeClient, namespace, "postgresql-operator", 1, retryInterval, timeout); err != nil {
		t.Fatal(err)
	}
	if err = postgreSQLClusterRestartTest(t, f, ctx); err != nil {
		t.Fatal(err)
	}
}

func postgreSQLClusterRestartTest(t *testing.T, f *framework.Framework, ctx *framework.TestCtx) error {
	namespace, err := ctx.GetNamespace()
	if err != nil {
		return fmt.Errorf("Couldn't get namespace: %v", err)
	}
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
		Priority:  0,
		Resources: resources,
		Storage:   postgresqlv1.PostgreSQLStorageSpec{},
	}

	examplePostgreSQL := &postgresqlv1.PostgreSQL{
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

	if err := f.Client.Create(goctx.TODO(), examplePostgreSQL, &framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval}); err != nil {
		return fmt.Errorf("Failed to create example PostgreSQL: %v", err)
	}
	if err := e2eutil.WaitForDeployment(t, f.KubeClient, namespace, "primary-node", 1, retryInterval, timeout); err != nil {
		return fmt.Errorf("Waiting for deployment primary-node timed out: %v", err)
	}
	if err := e2eutil.WaitForDeployment(t, f.KubeClient, namespace, "standby-node", 1, retryInterval, timeout); err != nil {
		return fmt.Errorf("Waiting for deployment standby-node timed out: %v", err)
	}
	if err := retryExecution(t, f, namespace, getStatusDouble, 7, time.Second*10); err != nil {
		return err
	}
	t.Log("Initial deployment created.")
	standbyListOpts := metav1.ListOptions{LabelSelector: "node-name=standby-node"}
	standbyPodList, err := f.KubeClient.CoreV1().Pods(namespace).List(standbyListOpts)
	if err != nil {
		return fmt.Errorf("Failed to get standby pod list: %v", err)
	}
	f.KubeClient.CoreV1().Pods(namespace).Delete(standbyPodList.Items[0].Name, &metav1.DeleteOptions{})
	if err := e2eutil.WaitForDeployment(t, f.KubeClient, namespace, "standby-node", 1, retryInterval, timeout); err != nil {
		return fmt.Errorf("Waiting for deployment standby-node timed out: %v", err)
	}
	if err := retryExecution(t, f, namespace, getStatusDouble, 7, time.Second*10); err != nil {
		return err
	}
	t.Log("Standby restart success.")

	primaryListOpts := metav1.ListOptions{LabelSelector: "node-name=primary-node"}
	primaryPodList, err := f.KubeClient.CoreV1().Pods(namespace).List(primaryListOpts)
	if err != nil {
		return fmt.Errorf("Failed to get primary pod list: %v", err)
	}
	f.KubeClient.CoreV1().Pods(namespace).Delete(primaryPodList.Items[0].Name, &metav1.DeleteOptions{})
	if err := e2eutil.WaitForDeployment(t, f.KubeClient, namespace, "primary-node", 1, retryInterval, timeout); err != nil {
		return fmt.Errorf("Waiting for deployment primary-node timed out: %v", err)
	}
	if err := retryExecution(t, f, namespace, getStatusDouble, 7, time.Second*10); err != nil {
		return err
	}
	t.Log("Primary restart success.")

	t.Log("Success")
	return nil
}
