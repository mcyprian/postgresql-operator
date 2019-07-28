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
)

const (
	retryInterval        = time.Second * 2
	timeout              = time.Second * 300
	cleanupRetryInterval = time.Second * 1
	cleanupTimeout       = time.Second * 5
	postgreSQLCRName     = "example-postgresql"
)

func TestPostgreSQL(t *testing.T) {
	postgreSQLList := &postgresqlv1.PostgreSQLList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PostgreSQL",
			APIVersion: postgresqlv1.SchemeGroupVersion.String(),
		},
	}
	err := framework.AddToFrameworkScheme(postgresqlv1.SchemeBuilder.AddToScheme, postgreSQLList)
	if err != nil {
		t.Fatalf("Failed to add custom resource scheme to framework: %v", err)
	}
	// run subtests
	t.Run("postgresql-group", func(t *testing.T) {
		t.Run("Cluster", PostgreSQLCluster)
	})
}

func PostgreSQLCluster(t *testing.T) {
	t.Parallel()
	ctx := framework.NewTestCtx(t)
	defer ctx.Cleanup()
	err := ctx.InitializeClusterResources(&framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval})
	if err != nil {
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
	err = e2eutil.WaitForDeployment(t, f.KubeClient, namespace, "postgresql-operator", 1, retryInterval, timeout)
	if err != nil {
		t.Fatal(err)
	}
	if err = postgreSQLClusterCreateTest(t, f, ctx); err != nil {
		t.Fatal(err)
	}
}

func postgreSQLClusterCreateTest(t *testing.T, f *framework.Framework, ctx *framework.TestCtx) error {
	namespace, err := ctx.GetNamespace()
	if err != nil {
		return fmt.Errorf("Couldn't get namespace: %v", err)
	}
	cpuValue, _ := resource.ParseQuantity("500m")
	memValue, _ := resource.ParseQuantity("1Gi")
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
	err = f.Client.Create(goctx.TODO(), examplePostgreSQL, &framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval})
	if err != nil {
		return fmt.Errorf("Failed to create example PostgreSQL: %v", err)
	}
	err = e2eutil.WaitForDeployment(t, f.KubeClient, namespace, "primary-node", 1, retryInterval, timeout)
	if err != nil {
		return fmt.Errorf("Waiting for deployment primary-node timed out: %v", err)
	}
	err = e2eutil.WaitForDeployment(t, f.KubeClient, namespace, "standby-node", 1, retryInterval, timeout)
	if err != nil {
		return fmt.Errorf("Waiting for deployment standby-node timed out: %v", err)
	}
	t.Log("Initial deployment created.")

	t.Log("Success")
	return nil
}
