# PostgreSQL Operator 

PostgreSQL operator to run replicated cluster on top Openshift and Kubernetes.
Operator uses [Operator Framework SDK](https://github.com/operator-framework/operator-sdk).

## Getting started

Setup Service Account, RBAC, CRD and deploy the operator:

    $ make up


### Create a PostgreSQL cluster
    
    $ oc apply -f example/example-postgresql.yaml

A three node cluster will be created:

    $ oc get pods

    NAME                                   READY     STATUS    RESTARTS   AGE
    node-one-86d9b546cc-7sls5               1/1	     Running   0          1m
    node-three-56c7fbbf6f-n758l             1/1	     Running   0          1m
    node-two-9dc447946-kscq7                1/1	     Running   0          1m
    postgresql-operator-78b4c4fbf7-nzj9k    1/1	     Running   0          2m


### Modify the cluster 

Modify the cluster definition:

    $ cat << EOF >> example/example-postgresql.yaml
          node-four:
            image: mcyprian/postgresql-10-fedora29:1.0
            priority: 30
            storage:
              storageClassName: local-storage
              size: 256Mi
      EOF

Apply changes to the cluster:

    $ oc apply -f example/example-postgresql.yaml

The fourth node will be created:

    $ oc get pods

    node-four-85f4b65f6-kzm87              1/1	 Running   0          28s
    node-one-6d99b78db9-dlr59              1/1	 Running   0          3m
    node-three-674c578b8-h4vlp             1/1	 Running   0          3m
    node-two-cd47c7d79-jknz7               1/1	 Running   0          3m
    postgresql-operator-78b4c4fbf7-kbglm   1/1	 Running   0          4m


### Basic monitoring of cluster status

    $ oc describe postgresql example-postgresql

    Name:         example-postgresql
    Namespace:    myproject
    ...
    Status:
    Nodes:
      Node - Four:
        Deployment Name:  node-four
        Pgversion:        10.10
        Priority:         30
        Role:             standby
        Service Name:     node-four
      Node - One:
        Deployment Name:  node-one
        Pgversion:        10.10
        Priority:         100
        Role:             primary
        Service Name:     node-one
      Node - Three:
        Deployment Name:  node-three
        Pgversion:        10.10
        Priority:         60
        Role:             standby
        Service Name:     node-three
      Node - Two:
        Deployment Name:  node-two
        Pgversion:        10.10
        Priority:         80
        Role:             standby
        Service Name:     node-two


### Destroy the cluster:

    $ oc delete -f example/example-postgresql.yaml


## Building operator locally

To rebuld operator run:

    $ make build


## Testing

Run unit e2e tests:

    $ make test
