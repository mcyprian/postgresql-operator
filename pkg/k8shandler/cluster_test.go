package k8shandler

import (
	"testing"

	postgresqlv1 "github.com/mcyprian/postgresql-operator/pkg/apis/postgresql/v1"
)

func TestGetHighestPriority(t *testing.T) {

	table := []struct {
		nodeMap  map[string]postgresqlv1.PostgreSQLNode
		expected string
	}{
		{
			map[string]postgresqlv1.PostgreSQLNode{
				"node-one": postgresqlv1.PostgreSQLNode{
					Priority: 100,
				},
				"node-two": postgresqlv1.PostgreSQLNode{
					Priority: 80,
				},
			}, "node-one"},
		{
			map[string]postgresqlv1.PostgreSQLNode{
				"node-one": postgresqlv1.PostgreSQLNode{
					Priority: 60,
				},
				"node-two": postgresqlv1.PostgreSQLNode{
					Priority: 60,
				},
				"node-three": postgresqlv1.PostgreSQLNode{
					Priority: 100,
				},
			}, "node-three"},
		{
			map[string]postgresqlv1.PostgreSQLNode{
				"node-one": postgresqlv1.PostgreSQLNode{
					Priority: 60,
				},
				"node-two": postgresqlv1.PostgreSQLNode{
					Priority: 60,
				},
				"node-three": postgresqlv1.PostgreSQLNode{
					Priority: 0,
				},
			}, "node-one"},
	}
	for _, tt := range table {
		actual, _, err := getHighestPriority(tt.nodeMap)
		if err != nil {
			t.Errorf("Test failed, err: %v", err)
		}
		if actual != tt.expected {
			t.Errorf("Test failed, expected: '%v', got: '%v'", tt.expected, actual)
		}
	}
}
