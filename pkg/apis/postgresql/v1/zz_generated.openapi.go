// +build !

// This file was autogenerated by openapi-gen. Do not edit it manually!

package v1

import (
	spec "github.com/go-openapi/spec"
	common "k8s.io/kube-openapi/pkg/common"
)

func GetOpenAPIDefinitions(ref common.ReferenceCallback) map[string]common.OpenAPIDefinition {
	return map[string]common.OpenAPIDefinition{
		"github.com/mcyprian/postgresql-operator/pkg/apis/postgresql/v1.PostgreSQL":       schema_pkg_apis_postgresql_v1_PostgreSQL(ref),
		"github.com/mcyprian/postgresql-operator/pkg/apis/postgresql/v1.PostgreSQLNode":   schema_pkg_apis_postgresql_v1_PostgreSQLNode(ref),
		"github.com/mcyprian/postgresql-operator/pkg/apis/postgresql/v1.PostgreSQLSpec":   schema_pkg_apis_postgresql_v1_PostgreSQLSpec(ref),
		"github.com/mcyprian/postgresql-operator/pkg/apis/postgresql/v1.PostgreSQLStatus": schema_pkg_apis_postgresql_v1_PostgreSQLStatus(ref),
	}
}

func schema_pkg_apis_postgresql_v1_PostgreSQL(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "PostgreSQL is the Schema for the postgresqls API",
				Properties: map[string]spec.Schema{
					"kind": {
						SchemaProps: spec.SchemaProps{
							Description: "Kind is a string value representing the REST resource this object represents. Servers may infer this from the endpoint the client submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#types-kinds",
							Type:        []string{"string"},
							Format:      "",
						},
					},
					"apiVersion": {
						SchemaProps: spec.SchemaProps{
							Description: "APIVersion defines the versioned schema of this representation of an object. Servers should convert recognized schemas to the latest internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#resources",
							Type:        []string{"string"},
							Format:      "",
						},
					},
					"metadata": {
						SchemaProps: spec.SchemaProps{
							Ref: ref("k8s.io/apimachinery/pkg/apis/meta/v1.ObjectMeta"),
						},
					},
					"spec": {
						SchemaProps: spec.SchemaProps{
							Ref: ref("github.com/mcyprian/postgresql-operator/pkg/apis/postgresql/v1.PostgreSQLSpec"),
						},
					},
					"status": {
						SchemaProps: spec.SchemaProps{
							Ref: ref("github.com/mcyprian/postgresql-operator/pkg/apis/postgresql/v1.PostgreSQLStatus"),
						},
					},
				},
			},
		},
		Dependencies: []string{
			"github.com/mcyprian/postgresql-operator/pkg/apis/postgresql/v1.PostgreSQLSpec", "github.com/mcyprian/postgresql-operator/pkg/apis/postgresql/v1.PostgreSQLStatus", "k8s.io/apimachinery/pkg/apis/meta/v1.ObjectMeta"},
	}
}

func schema_pkg_apis_postgresql_v1_PostgreSQLNode(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "PostgreSQLNode defines individual node in PostgreSQL cluster",
				Properties: map[string]spec.Schema{
					"image": {
						SchemaProps: spec.SchemaProps{
							Type:   []string{"string"},
							Format: "",
						},
					},
					"priority": {
						SchemaProps: spec.SchemaProps{
							Type:   []string{"integer"},
							Format: "int32",
						},
					},
					"resources": {
						SchemaProps: spec.SchemaProps{
							Ref: ref("k8s.io/api/core/v1.ResourceRequirements"),
						},
					},
					"storage": {
						SchemaProps: spec.SchemaProps{
							Ref: ref("github.com/mcyprian/postgresql-operator/pkg/apis/postgresql/v1.PostgreSQLStorageSpec"),
						},
					},
				},
				Required: []string{"priority", "storage"},
			},
		},
		Dependencies: []string{
			"github.com/mcyprian/postgresql-operator/pkg/apis/postgresql/v1.PostgreSQLStorageSpec", "k8s.io/api/core/v1.ResourceRequirements"},
	}
}

func schema_pkg_apis_postgresql_v1_PostgreSQLSpec(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "PostgreSQLSpec defines the desired state of PostgreSQL",
				Properties: map[string]spec.Schema{
					"managementState": {
						SchemaProps: spec.SchemaProps{
							Type:   []string{"string"},
							Format: "",
						},
					},
					"nodes": {
						SchemaProps: spec.SchemaProps{
							Type: []string{"object"},
							AdditionalProperties: &spec.SchemaOrBool{
								Schema: &spec.Schema{
									SchemaProps: spec.SchemaProps{
										Ref: ref("github.com/mcyprian/postgresql-operator/pkg/apis/postgresql/v1.PostgreSQLNode"),
									},
								},
							},
						},
					},
				},
				Required: []string{"managementState", "nodes"},
			},
		},
		Dependencies: []string{
			"github.com/mcyprian/postgresql-operator/pkg/apis/postgresql/v1.PostgreSQLNode"},
	}
}

func schema_pkg_apis_postgresql_v1_PostgreSQLStatus(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "PostgreSQLStatus defines the observed state of PostgreSQL",
				Properties: map[string]spec.Schema{
					"nodes": {
						SchemaProps: spec.SchemaProps{
							Type: []string{"object"},
							AdditionalProperties: &spec.SchemaOrBool{
								Schema: &spec.Schema{
									SchemaProps: spec.SchemaProps{
										Ref: ref("github.com/mcyprian/postgresql-operator/pkg/apis/postgresql/v1.PostgreSQLNodeStatus"),
									},
								},
							},
						},
					},
				},
				Required: []string{"nodes"},
			},
		},
		Dependencies: []string{
			"github.com/mcyprian/postgresql-operator/pkg/apis/postgresql/v1.PostgreSQLNodeStatus"},
	}
}
