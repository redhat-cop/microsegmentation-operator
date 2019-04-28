# Microsegmentation Operator

[![Build Status](https://travis-ci.org/redhat-cop/microsegmentation-operator.svg?branch=master)](https://travis-ci.org/redhat-cop/microsegmentation-operator) [![Docker Repository on Quay](https://quay.io/repository/redhat-cop/microsegmentation-operator/status "Docker Repository on Quay")](https://quay.io/repository/redhat-cop/microsegmentation-operator)

The microsegmentation operator allows to create [NetworkPolicy]() rules starting from [Services]().
This feature is activated by this annotation: `microsegmentation-operator.redhat-cop.io/microsegmentation: "true"`
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

Inbound/outbound ports and in AND with corresponding inbound/outbound pod label selectors and namespace label selectors.

Pod label selectors and namespace label selectors are in OR with each other.

## Deploying the Operator

This is a cluster-level operator that you can deploy in any namespace, `microsegmentation-operator` is recommeded.

```shell
oc new-project microsegmentation-operator
```

Deploy the cluster resources. Given that a number of elevated permissions are required to resources at a cluster scope the account you are currently logged in must have elevated rights.

```shell
oc apply -f deploy
```

## Local Development

Execute the following steps to develop the functionality locally. It is recommended that development be done using a cluster with `cluster-admin` permissions.

Clone the repository, then resolve all depdendencies using `dep`:

```shell
dep ensure
```

Using the [operator-sdk](https://github.com/operator-framework/operator-sdk), run the operator locally:

```shell
operator-sdk up local --namespace ""
```