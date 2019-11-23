package e2e

import (
	"fmt"
	"testing"
	"time"

	goctx "context"

	postgresqlv1 "github.com/mcyprian/postgresql-operator/pkg/apis/postgresql/v1"

	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"k8s.io/apimachinery/pkg/types"
)

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
			if status.Role != "primary" || status.Priority != 100 {
				return fmt.Errorf("Wrong node role or status, expected (%v, %v), got: (%v, %v)", "primary", 100, status.Role, status.Priority)
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
			if status.Role != "primary" || status.Priority != 80 {
				return fmt.Errorf("Wrong node role or status, expected (%v, %v), got: (%v, %v)", "primary", 80, status.Role, status.Priority)
			}
		} else {
			return fmt.Errorf("Wrong node name %v, only standby-node should be present in the cluster", name)
		}
	}
	return nil
}
