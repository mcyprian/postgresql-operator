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
