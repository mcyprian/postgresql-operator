package k8shandler

import (
	postgresqlv1 "github.com/mcyprian/postgresql-operator/pkg/apis/postgresql/v1"
	"testing"
)

var getHighestTests = []struct {
	in       map[string]postgresqlv1.PostgreSQLNode
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

func TestGetHighestPriority(t *testing.T) {
	for _, tt := range getHighestTests {
		actual, _, err := getHighestPriority(tt.in)
		if err != nil {
			t.Errorf("Test failed, err: %d", err)
		}
		if actual != tt.expected {
			t.Errorf("Test failed, expected: '%s', got: '%s'", tt.expected, actual)
		}
	}
}
