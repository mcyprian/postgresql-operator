package e2e

import (
	"testing"
	"time"

	postgresqlv1 "github.com/mcyprian/postgresql-operator/pkg/apis/postgresql/v1"
	framework "github.com/operator-framework/operator-sdk/pkg/test"
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
	if err := framework.AddToFrameworkScheme(postgresqlv1.SchemeBuilder.AddToScheme, postgreSQLList); err != nil {
		t.Fatalf("Failed to add custom resource scheme to framework: %v", err)
	}
	// run subtests
	t.Run("postgresql-group", func(t *testing.T) {
		t.Run("ClusterScaling", PostgreSQLClusterScaling)
		t.Run("ClusterFailover", PostgreSQLClusterFailover)
	})
}
