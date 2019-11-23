package k8shandler

import (
	"os"
	"testing"
)

func TestNewPgPasswords(t *testing.T) {
	os.Unsetenv("POSTGRESQL_PASSWORD")
	actual, _ := newPgPasswords()
	if len(actual.repmgr) != 32 || len(actual.database) != 32 {
		t.Errorf("Test failed, generated passwords don't have an expected length.")
	}
	os.Setenv("POSTGRESQL_PASSWORD", "secrettestpassword")
	actual, _ = newPgPasswords()
	if actual.database != "secrettestpassword" {
		t.Errorf("Test failed, database password doesn't reflect the env variable.")
	}
}
