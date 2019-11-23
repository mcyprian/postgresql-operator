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
	"k8s.io/apimachinery/pkg/types"
)

// PostgreSQLClusterScaling test downscaling of slave node and rejoining the cluster again
func PostgreSQLClusterScaling(t *testing.T) {
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
	if err = postgreSQLClusterScalingTest(t, f, ctx); err != nil {
		t.Fatal(err)
	}
}

func postgreSQLClusterScalingTest(t *testing.T, f *framework.Framework, ctx *framework.TestCtx) error {
	namespace, err := ctx.GetNamespace()
	if err != nil {
		return fmt.Errorf("Couldn't get namespace: %v", err)
	}
	exampleName := types.NamespacedName{Name: postgreSQLCRName, Namespace: namespace}
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
	standbyDeployment, err := f.KubeClient.AppsV1().Deployments(namespace).Get("standby-node", metav1.GetOptions{IncludeUninitialized: true})
	if err != nil {
		return fmt.Errorf("Failed to get standby-node deployment: %v", err)
	}
	current := &postgresqlv1.PostgreSQL{}
	if err := f.Client.Get(goctx.TODO(), exampleName, current); err != nil {
		return fmt.Errorf("Failed to get examplePostgreSQL: %v", err)
	}
	delete(current.Spec.Nodes, "standby-node")

	if err := f.Client.Update(goctx.TODO(), current); err != nil {
		return fmt.Errorf("Failed to update cluster: %v", err)
	}
	if err := e2eutil.WaitForDeletion(t, f.Client.Client, standbyDeployment, retryInterval, timeout); err != nil {
		return fmt.Errorf("Waiting for standby-node deletion timed out: %v", err)
	}
	if err = retryExecution(t, f, namespace, getStatusSingle, 7, time.Second*10); err != nil {
		return err
	}
	t.Log("Downscale success.")
	if err := f.Client.Get(goctx.TODO(), exampleName, current); err != nil {
		return fmt.Errorf("Failed to get examplePostgreSQL: %v", err)
	}
	current.Spec.Nodes["standby-node"] = standbyNode

	if err := f.Client.Update(goctx.TODO(), current); err != nil {
		return fmt.Errorf("Failed to update cluster: %v", err)
	}

	if err := e2eutil.WaitForDeployment(t, f.KubeClient, namespace, "standby-node", 1, retryInterval, timeout); err != nil {
		return fmt.Errorf("Waiting for deployment standby-node timed out: %v", err)
	}
	if err = retryExecution(t, f, namespace, getStatusDouble, 7, time.Second*10); err != nil {
		return err
	}
	t.Log("Upscale success.")

	t.Log("Success")
	return nil
}
