# Microsegmentation Operator

[![Build Status](https://travis-ci.org/redhat-cop/microsegmentation-operator.svg?branch=master)](https://travis-ci.org/redhat-cop/microsegmentation-operator) [![Docker Repository on Quay](https://quay.io/repository/redhat-cop/microsegmentation-operator/status "Docker Repository on Quay")](https://quay.io/repository/redhat-cop/microsegmentation-operator)

The microsegmentation operator allows to create [NetworkPolicies](https://kubernetes.io/docs/concepts/services-networking/network-policies/) rules starting from [Services](https://kubernetes.io/docs/concepts/services-networking/service/).

This feature is activated by this annotation: `microsegmentation-operator.redhat-cop.io/microsegmentation: "true"`.

By default the generated NetworkPolicy will allow traffic from pods in the same namespace and to the ports described in the service.

The NetworkPolicy object can be tweaked with the following additional annotations:

| Annotation  | Description  |
| - | - |
| `microsegmentation-operator.redhat-cop.io/additional-inbound-ports`  | comma separated list of allowed inbound ports expressed in this format: *port/protocol*; e.g. `8888/TCP,9999/UDP`  |
|  `microsegmentation-operator.redhat-cop.io/inbound-pod-labels` | comma separated list of labels to be used as label selectors for allowed inbound pods; e.g. `label1=value1,label2=value2`  |
| `microsegmentation-operator.redhat-cop.io/inbound-namespace-labels`  | comma separated list of labels to be used as label selectors for allowed inbound namespaces; e.g. `label1=value1,label2=value2`  |
| `microsegmentation-operator.redhat-cop.io/outbound-pod-labels`  | comma separated list of labels to be used as label selectors for allowed outbound pods; e.g. `label1=value1,label2=value2`  ||   |   |
| `microsegmentation-operator.redhat-cop.io/outbound-namespace-labels`  | comma separated list of labels to be used as label selectors for allowed outbound namespaces; e.g. `label1=value1,label2=value2`  |
| `microsegmentation-operator.redhat-cop.io/outbound-ports`  | comma separated list of allowed outbound ports expressed in this format: *port/protocol*; e.g. `8888/TCP,9999/UDP`  |

Inbound/outbound ports and in `AND` with corresponding inbound/outbound pod label selectors and namespace label selectors.

The pod label selector and the namespace label selector are in OR with each other.

It should be relatively common to use the `additional-inbound-ports` annotation to model those situation where a pod exposes a port that should not be load balanced.

All the other annotation are there to provide flexibility, but should not be used extensively. If you find yourself making high use of them, you have probably reached the point where you should create NetworkPolicies directly.

## Examples

The following service:

```yaml
apiVersion: v1
kind: Service
metadata:
  annotations:
    microsegmentation-operator.redhat-cop.io/microsegmentation: "true"
  name: test1
spec:
  ports:
  - name: https
    port: 443
    protocol: TCP
    targetPort: 8443
  - name: https1
    port: 4431
    protocol: TCP
    targetPort: 8431
  selector:
    app: console
    component: ui
```

produces the following NetworkPolicy:

```yaml
apiVersion: extensions/v1beta1
kind: NetworkPolicy
metadata:
  name: test1
spec:
  ingress:
  - from:
    - podSelector: {}
    ports:
    - port: 8443
      protocol: TCP
    - port: 8431
      protocol: TCP
  podSelector:
    matchLabels:
      app: console
      component: ui
  policyTypes:
  - Ingress
```

The following service:

```yaml
apiVersion: v1
kind: Service
metadata:
  annotations:
    microsegmentation-operator.redhat-cop.io/microsegmentation: "true"
    microsegmentation-operator.redhat-cop.io/additional-inbound-ports: 123/TCP,456/UDP
    microsegmentation-operator.redhat-cop.io/inbound-pod-labels: app=gateway,application=3scale
    microsegmentation-operator.redhat-cop.io/inbound-namespace-labels: frontend=abc,frontend-user=customers
    microsegmentation-operator.redhat-cop.io/outbound-pod-labels: app=database,application=db2
    microsegmentation-operator.redhat-cop.io/outbound-namespace-labels: backend=cfg,backend-user=internal
    microsegmentation-operator.redhat-cop.io/outbound-ports: 789/TCP,012/UDP
  name: test2
spec:
  ports:
  - name: https
    port: 443
    protocol: TCP
    targetPort: 8443
  - name: https1
    port: 4431
    protocol: TCP
    targetPort: 8431
  selector:
    app: console
    component: ui
```

produces the following NetworkPolicy:

```yaml
apiVersion: extensions/v1beta1
kind: NetworkPolicy
metadata:
  generation: 2
  name: test2
spec:
  egress:
  - ports:
    - port: 789
      protocol: TCP
    - port: 12
      protocol: UDP
    to:
    - podSelector:
        matchLabels:
          app: database
          application: db2
  - ports:
    - port: 789
      protocol: TCP
    - port: 12
      protocol: UDP
    to:
    - namespaceSelector:
        matchLabels:
          backend: cfg
          backend-user: internal
  ingress:
  - from:
    - podSelector:
        matchLabels:
          app: gateway
          application: 3scale
    ports:
    - port: 8443
      protocol: TCP
    - port: 8431
      protocol: TCP
    - port: 123
      protocol: TCP
    - port: 456
      protocol: UDP
  - from:
    - namespaceSelector:
        matchLabels:
          frontend: abc
          frontend-user: customers
    ports:
    - port: 8443
      protocol: TCP
    - port: 8431
      protocol: TCP
    - port: 123
      protocol: TCP
    - port: 456
      protocol: UDP
  podSelector:
    matchLabels:
      app: console
      component: ui
  policyTypes:
  - Ingress
  - Egress
```

## Deploying the Operator

This is a cluster-level operator that you can deploy in any namespace, `microsegmentation-operator` is recommended.

```shell
oc new-project microsegmentation-operator
```

Deploy the cluster resources. Given that a number of elevated permissions are required to resources at a cluster scope the account you are currently logged in must have elevated rights.

```shell
oc apply -f deploy
```

## Local Development

Execute the following steps to develop the functionality locally. It is recommended that development be done using a cluster with `cluster-admin` permissions.

```shell
go mod download
```

optionally:

```shell
go mod vendor
```

Using the [operator-sdk](https://github.com/operator-framework/operator-sdk), run the operator locally:

```shell
OPERATOR_NAME='microsegmentation-operator' operator-sdk --verbose up local --namespace ""
```

## Release Process

To release execute the following:

```shell
git tag -a "<version>" -m "release <version>"
git push upstream <version>
```

use this version format: vM.m.z
