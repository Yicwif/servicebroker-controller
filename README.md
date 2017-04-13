# ServiceBroker Controller

## Goal

The goal of this project is to demonstrate how you can build a simple Kubernetes controller.

This is not meant as a project to be used directly - but rather as a reference point to build your own custom controllers.

This example is currently based off client-go v2.0.0 - but will be updated as new versions become available.

## Helpful Resources

- github.com/kubernetes/community
    - contributors/devel/controllers.md
    - contributors/design-proposals/principles.md#control-logic

- github.com/kubernetes/kubernetes
    - pkg/controller

- github.com/kubernetes/client-go
    - examples/  (Note: examples are version sensitive)

- github.com/kbst/memcached
    - Operator written in Python

## Roadmap

- Update to client-go v3.0.0 (when available)
- Demonstrate using leader-election
- Demonstrate using work-queues
- Demonstrate using Third Party Resources
- Demonstrate using Shared Informers

## Building

Build agent and controller binaries:

`make clean all`

Build agent and controller Docker images:

`make clean images`

------------

## Production setup

For production, we recommend users to limit access to only the resources operator needs, and create a specific role, service account for operator.

### Create ClusterRole

We will use ClusterRole instead of Role because etcd operator accesses non-namespaced resources, e.g. Third Party Resource.

Create the following ClusterRole

```bash
$ cat <<EOF | kubectl create -f -
apiVersion: rbac.authorization.k8s.io/v1beta1
kind: ClusterRole
metadata:
  name: etcd-operator
rules:
- apiGroups:
  - etcd.coreos.com
  resources:
  - clusters
  verbs:
  - "*"
- apiGroups:
  - extensions
  resources:
  - thirdpartyresources
  verbs:
  - create
- apiGroups:
  - storage.k8s.io
  resources:
  - storageclasses
  verbs:
  - create
- apiGroups: 
  - ""
  resources:
  - pods
  - services
  - endpoints
  - persistentvolumeclaims
  verbs:
  - "*"
- apiGroups:
  - extensions
  resources:
  - replicasets
  verbs:
  - "*"
EOF
```

If you need use s3 backup, add these to above input:

```
- apiGroups: 
  - ""
  resources: 
  - secrets
  - configmaps
  verbs:
  - get
```

### Create Service Account

Modify or export env `ETCD_OPERATOR_NS` to your current namespace, 
and create ServiceAccount for etcd operator:

```bash
$ cat <<EOF | kubectl create -f -
apiVersion: v1
kind: ServiceAccount
metadata:
  name: etcd-operator
  namespace: $ETCD_OPERATOR_NS
EOF
```

### Create ClusterRoleBinding

Modify or export env `ETCD_OPERATOR_NS` to your current namespace, 
and create ClusterRoleBinding for etcd operator:

```bash
$ cat <<EOF | kubectl create -f -
apiVersion: rbac.authorization.k8s.io/v1beta1
kind: ClusterRoleBinding
metadata:
  name: etcd-operator
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: etcd-operator
subjects:
- kind: ServiceAccount
  name: etcd-operator
  namespace: $ETCD_OPERATOR_NS
EOF
```

### Run deployment with service account

For etcd operator pod or deployment, fill the pod template with service account `etcd-operator` created above.

For example:

```yaml
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: etcd-operator
spec:
  template:
    spec:
      serviceAccountName: etcd-operator
      ...
```
