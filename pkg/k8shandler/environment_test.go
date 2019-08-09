package k8shandler

import (
	"os"
	"testing"
)

func TestNewPgEnvironment(t *testing.T) {
	table := []struct {
		osEnv    map[string]string
		expected pgEnvironment
	}{
		{
			map[string]string{},
			pgEnvironment{user: defaultPgUser, database: defaultPgDatabase},
		},
		{
			map[string]string{"POSTGRESQL_USER": "test"},
			pgEnvironment{user: "test", database: defaultPgDatabase},
		},
		{
			map[string]string{"POSTGRESQL_USER": "testuser",
				"POSTGRESQL_DATABASE": "testdb",
			},
			pgEnvironment{user: "testuser", database: "testdb"},
		},
	}
	for _, tt := range table {
		for name, value := range tt.osEnv {
			if err := os.Setenv(name, value); err != nil {
				t.Errorf("Test failed, err: %v", err)
			}
		}
		actual := newPgEnvironment()
		if actual != tt.expected {
			t.Errorf("Test failed, expected: '%v', got: '%v'", tt.expected, actual)
		}
	}
}
