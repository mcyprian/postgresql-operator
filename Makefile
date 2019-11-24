OC=oc
DEPLOYMENT_TARGETS= \
	deploy/crds/postgresql_v1_postgresql_crd.yaml \
	deploy/operator.yaml \
	deploy/role_binding.yaml \
	deploy/role.yaml \
	deploy/service_account.yaml

OPERATOR_IMAGE=mcyprian/postgresql-operator
OPERATOR_VERSION=v0.0.1

.PHONY: all linters test build push up down test-e2e test-unit fmt imports ensure-imports lint

all: build push

linters: fmt vet ensure-imports lint

test: test-unit test-e2e

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

test-e2e:
	@operator-sdk test local ./test/e2e --go-test-flags "-v -parallel=1 --timeout 20m"

test-unit:
	@go test -v ./pkg/... ./cmd/...

fmt:
	@gofmt -l -w cmd && \
	gofmt -l -w pkg && \
	gofmt -l -w test

vet:
	@go vet ./...

lint:
	@golint -set_exit_status pkg/k8shandler && \
	golint -set_exit_status pkg/apis && \
	golint -set_exit_status test/e2e 

imports:
	@goimports -w cmd pkg test

ensure-imports:
	@if [ $$(goimports -l pkg test | wc -l) -ne 0 ]; \
	then echo "Formatting of some files differs from goimport's"; false; \
	fi

mod:
	@go mod tidy && \
	go mod verify
