package k8shandler

import (
	"os"
)

type pgEnvironment struct {
	user     string
	database string
}

func newPgEnvironment() pgEnvironment {
	user := os.Getenv("POSTGRESQL_USER")
	if user == "" {
		user = defaultPgUser
	}
	database := os.Getenv("POSTGRESQL_DATABASE")
	if database == "" {
		database = defaultPgDatabase
	}
	return pgEnvironment{
		user:     user,
		database: database,
	}
}
