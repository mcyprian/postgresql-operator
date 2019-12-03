package k8shandler

const (
	postgresqlPort = 5432

	defaultPgImage            = "mcyprian/postgresql-10-fedora29:1.0"
	defaultPgUser             = "user"
	defaultPgDatabase         = "db"
	defaultCntCommand         = "run-repmgr-replica"
	defaultHealthCheckCommand = "/usr/libexec/check-container"

	defaultCPULimit      = "400m"
	defaultCPURequest    = "100m"
	defaultMemoryLimit   = "1Gi"
	defaultMemoryRequest = "500Mi"

	pgDataPath     = "/var/lib/pgsql/data/"
	pgpassFilePath = "/var/lib/pgsql/.pgpass"
)
