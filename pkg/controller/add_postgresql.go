package controller

import (
	"github.com/mcyprian/postgresql-operator/pkg/controller/postgresql"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, postgresql.Add)
}
