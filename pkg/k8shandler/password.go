package k8shandler

import (
	"fmt"
	"os"

	"github.com/sethvargo/go-password/password"
)

type pgPasswords struct {
	repmgr   string
	database string
}

func newPgPasswords() (*pgPasswords, error) {
	repmgr, err := generatePassword()
	if err != nil {
		return nil, fmt.Errorf("Failed to generate password for repmgr user: %v", err)
	}
	database := os.Getenv("POSTGRESQL_PASSWORD")
	if database == "" {
		database, err = generatePassword()
		if err != nil {
			return nil, fmt.Errorf("Failed to generate password for database user: %v", err)
		}
	}
	return &pgPasswords{
		repmgr:   repmgr,
		database: database,
	}, nil
}

// generatePassword generates high-entropy random password, 32 characters long, 5 digits, 5 symbols
// including upper and lower case letters
func generatePassword() (string, error) {
	return password.Generate(32, 5, 0, false, false)
}
