package k8shandler

import (
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/lib/pq"
	postgresqlv1 "github.com/mcyprian/postgresql-operator/pkg/apis/postgresql/v1"
)

type databaseInfo struct {
	host     string
	port     int
	user     string
	dbname   string
	password string
	sslmode  string
}

type database struct {
	info      databaseInfo
	engine    *sql.DB
	cachedErr error
}

func newRepmgrDatabase(host string) *database {
	return &database{
		info: databaseInfo{
			host:    host,
			port:    postgresqlPort,
			user:    "repmgr",
			dbname:  "repmgr",
			sslmode: "disable",
		},
		cachedErr: nil,
	}
}

func (info *databaseInfo) connectionString() string {
	return fmt.Sprintf("host=%s port=%d user=%s dbname=%s sslmode=%s",
		info.host, info.port, info.user, info.dbname, info.sslmode)
}

func (db *database) err() error {
	err := db.cachedErr
	db.cachedErr = nil
	return err
}

func (db *database) initialize() {
	if db.cachedErr == nil {
		db.engine, db.cachedErr = sql.Open("postgres", db.info.connectionString())
	}
}

func (db *database) ping() error {
	if db.cachedErr != nil {
		return db.cachedErr
	}

	return db.engine.Ping()
}

// version returns version of PostgreSQL server
func (db *database) version() string {
	var version string

	if db.cachedErr != nil {
		return "unknown"
	}

	row := db.engine.QueryRow("SELECT version()")
	if db.cachedErr = row.Scan(&version); db.cachedErr != nil {
		return "unknown"
	}
	fields := strings.Fields(version)
	if len(fields) <= 2 {
		db.cachedErr = fmt.Errorf("Failed to retrieve PostgreSQL version")
		return "unknown"
	}
	return fields[1]
}

// repmgrNodesExists checks whether repmgr.nodes table was created already
func (db *database) repmgrNodesExists() bool {
	var result bool

	if db.cachedErr != nil {
		return false
	}

	row := db.engine.QueryRow("SELECT EXISTS ( SELECT 1 FROM information_schema.tables WHERE table_schema = 'repmgr' AND table_name = 'nodes')")
	if db.cachedErr = row.Scan(&result); db.cachedErr != nil {
		return false
	}
	return result
}

// isRegistered checks whether node name is present in repmgr.nodes table
func (db *database) isRegistered(nodeName string) bool {
	var result int
	var stmt *sql.Stmt

	if db.cachedErr != nil {
		return false
	}

	// if repmgr.nodes table is missing, node is not registered
	exists := db.repmgrNodesExists()
	if db.cachedErr != nil || !exists {
		return false
	}

	// repmgr.nodes table exists, check if it contains row for the node
	stmt, db.cachedErr = db.engine.Prepare("SELECT COUNT(*) FROM repmgr.nodes WHERE node_name = $1")
	if db.cachedErr != nil {
		return false
	}
	if db.cachedErr = stmt.QueryRow(nodeName).Scan(&result); db.cachedErr != nil {
		return false
	}
	return result == 1
}

// getRole retrieves current node role inside repmgr cluster
func (db *database) getRole(nodeName string) postgresqlv1.PostgreSQLNodeRole {
	var result string
	var stmt *sql.Stmt

	if db.cachedErr != nil {
		return postgresqlv1.PostgreSQLNodeRoleUnknown
	}

	// if repmgr.nodes table is missing, role is unknown
	exists := db.repmgrNodesExists()
	if db.cachedErr != nil || !exists {
		return postgresqlv1.PostgreSQLNodeRoleUnknown
	}

	// repmgr.nodes table exists, check the role
	stmt, db.cachedErr = db.engine.Prepare("SELECT type FROM repmgr.nodes WHERE node_name = $1")
	if db.cachedErr != nil {
		return postgresqlv1.PostgreSQLNodeRoleUnknown
	}
	if db.cachedErr = stmt.QueryRow(nodeName).Scan(&result); db.cachedErr != nil {
		return postgresqlv1.PostgreSQLNodeRoleUnknown
	}
	if result == postgresqlv1.PostgreSQLNodeRolePrimary {
		return postgresqlv1.PostgreSQLNodeRolePrimary
	}
	return postgresqlv1.PostgreSQLNodeRoleStandby
}
