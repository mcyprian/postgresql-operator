package k8shandler

import (
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/lib/pq"
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
	info   databaseInfo
	engine *sql.DB
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
	}
}

func (info *databaseInfo) connectionString() string {
	return fmt.Sprintf("host=%s port=%d user=%s dbname=%s sslmode=%s",
		info.host, info.port, info.user, info.dbname, info.sslmode)
}

func (db *database) initialize() error {
	var err error
	db.engine, err = sql.Open("postgres", db.info.connectionString())
	if err != nil {
		return err
	}
	return nil
}

func (db *database) ping() error {
	return db.engine.Ping()
}

func (db *database) version() (string, error) {
	var version string
	row := db.engine.QueryRow("SELECT version()")
	if err := row.Scan(&version); err != nil {
		return "", err
	}
	fields := strings.Fields(version)
	if len(fields) <= 2 {
		return "", fmt.Errorf("Failed to retrieve PostgreSQL version")
	}
	return fields[1], nil
}

// repmgrNodesExists checks whether repmgr.nodes table was created already
func (db *database) repmgrNodesExists() (bool, error) {
	var result bool
	row := db.engine.QueryRow("SELECT EXISTS ( SELECT 1 FROM information_schema.tables WHERE table_schema = 'repmgr' AND table_name = 'nodes')")
	if err := row.Scan(&result); err != nil {
		return false, err
	}
	return result, nil
}

// isRegistered checks whether node name is present in repmgr.nodes table
func (db *database) isRegistered(nodeName string) (bool, error) {
	var result int

	// if repmgr.nodes table is missing, node is not registered
	exists, err := db.repmgrNodesExists()
	if err != nil {
		return false, err
	}
	if !exists {
		return false, nil
	}

	// repmgr.nodes table exists, check if it contains row for the node
	query, err := db.engine.Prepare("SELECT COUNT(*) FROM repmgr.nodes WHERE node_name = $1")
	if err != nil {
		return false, err
	}
	if err := query.QueryRow(nodeName).Scan(&result); err != nil {
		return false, err
	}
	return result == 1, nil
}
