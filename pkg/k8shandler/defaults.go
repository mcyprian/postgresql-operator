package k8shandler

const (
	postgresqlPort            = 5432
	defaultPgImage            = "mcyprian/postgresql-10-fedora29"
	defaultCntCommand         = "statefulset-startup"
	defaultHealthCheckCommand = "/usr/libexec/check-container"
)
