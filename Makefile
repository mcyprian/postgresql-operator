OC=oc
DEPLOYMENT_TARGETS= \
	deploy/crds/postgresql_v1_postgresql_crd.yaml \
	deploy/crds/postgresql_v1_postgresql_cr.yaml \
	deploy/operator.yaml \
	deploy/role_binding.yaml \
	deploy/role.yaml \
	deploy/service_account.yaml

OPERATOR_IMAGE=mcyprian/postgresql-operator
OPERATOR_VERSION=v0.0.1

.PHONY: all build push up down fmt

all: build push

build:
	@operator-sdk build $(OPERATOR_IMAGE):$(OPERATOR_VERSION)

push:
	@docker push $(OPERATOR_IMAGE):$(OPERATOR_VERSION)

up:
	@- $(foreach T,$(DEPLOYMENT_TARGETS), \
	$(OC) apply -f $T ;\
	)

down:
	@- $(foreach T,$(DEPLOYMENT_TARGETS), \
	$(OC) delete -f $T ;\
	)

test-unit:
	@go test -v ./pkg/... ./cmd/...

fmt:
	@gofmt -l -w cmd && \
	gofmt -l -w pkg

dep:
	dep ensure -v
