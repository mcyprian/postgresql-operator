package e2e

import (
	goctx "context"
	"fmt"
	"testing"
	"time"

	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/operator-framework/operator-sdk/pkg/test/e2eutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PostgreSQLClusterRestart test correct recovery of the cluster after slave and master node restart
func PostgreSQLClusterRestart(t *testing.T) {
	t.Parallel()
	ctx := framework.NewTestCtx(t)
	// get global framework variables reference
	f := framework.Global
	defer ctx.Cleanup()

	initializeTestEnvironment(t, f, ctx)

	if err := postgreSQLClusterRestartTest(t, f, ctx); err != nil {
		t.Fatal(err)
	}
}

func postgreSQLClusterRestartTest(t *testing.T, f *framework.Framework, ctx *framework.TestCtx) error {
	namespace, err := ctx.GetNamespace()
	if err != nil {
		return fmt.Errorf("Couldn't get namespace: %v", err)
	}
	examplePostgreSQL := newTestCluster(namespace)

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
