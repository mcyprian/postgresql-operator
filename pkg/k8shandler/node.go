package k8shandler

import (
	postgresqlv1 "github.com/mcyprian/postgresql-operator/pkg/apis/postgresql/v1"
)

// Node interface represents a single PostgreSQL node in the cluster
type Node interface {
	name() string
	create(request *PostgreSQLRequest) error
	update(request *PostgreSQLRequest, specNode *postgresqlv1.PostgreSQLNode, writableDB *database) (bool, error)
	delete(request *PostgreSQLRequest) error
	status() postgresqlv1.PostgreSQLNodeStatus
	dbClient() *database
	isRegistered(request *PostgreSQLRequest) (bool, error)
	isReady() bool
}
