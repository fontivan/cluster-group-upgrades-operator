# Topology Aware Lifecycle Manager (TALM) on OLM v1

TALM (cluster-group-upgrades-operator) is packaged for OLM v1 with **AllNamespaces** as the only supported install mode. The controller uses cluster-scoped RBAC and watches resources across the cluster; it does not set `spec.config.inline.watchNamespace` on the ClusterExtension (that field is invalid for AllNamespaces-only bundles).

## Prerequisites

- OpenShift cluster with [OLM v1 enabled](https://docs.redhat.com/en/documentation/openshift_container_platform/latest/html/extensions/olmv1-enabling-features)
- Advanced Cluster Management (ACM) 2.9+ on the hub (unchanged from classic install)
- `oc`, `jq`, and optionally `opm` for catalog inspection

## Reference CRs (RDS)

Install manifests live under [`deploy/olmv1/`](../../deploy/olmv1/):

| Resource | Purpose |
|----------|---------|
| `namespace.yaml` | Operator namespace (`openshift-cluster-group-upgrades`) |
| `serviceaccount.yaml` | OLM v1 installer ServiceAccount |
| `installer-clusterrolebinding.yaml` | Installer RBAC (reference uses `cluster-admin`; tighten per environment) |
| `imagedigestmirrorset.yaml` | Konflux image mirrors for dev/CI |
| `clustercatalog.yaml` | File-based catalog (FBC) source |
| `clusterextension.yaml` | OLM v1 install CR for package `topology-aware-lifecycle-manager` |

Apply everything:

```bash
oc apply -k deploy/olmv1/
```

Or use the scripted flow (waits for catalog/extension readiness):

```bash
chmod +x hack/olmv1/*.sh
./hack/olmv1/install.sh
./hack/olmv1/verify-install.sh
```

Environment variables for `install.sh`:

| Variable | Default |
|----------|---------|
| `TALM_NAMESPACE` | `openshift-cluster-group-upgrades` |
| `FBC_IMAGE` | `quay.io/redhat-user-workloads/telco-5g-tenant/topology-aware-lifecycle-manager-fbc-5-0:latest` |
| `CLUSTER_EXTENSION_VERSION` | `5.0.0` |
| `CLUSTER_EXTENSION_CHANNEL` | `stable` |
| `SKIP_IDMS` | `false` |

## Bundle / operator compatibility

Automated checks (no cluster required):

```bash
make olmv1-bundle-validate
```

Validates `registry+v1` bundle format, AllNamespaces-only install modes, no CSV webhooks, and no `olm.*.required` FBC dependency properties.

## Resource utilization (OwnNamespace vs AllNamespaces)

See [RESOURCE_UTILIZATION.md](./RESOURCE_UTILIZATION.md) for methodology and the results table. Collect samples with:

```bash
SCENARIO_LABEL=all-namespaces ./hack/olmv1/measure-resource-utilization.sh
```

## CI

- **Local / Prow `ci-job`**: `make olmv1-bundle-validate` and `go test ./tests/olmv1/...`
- **Cluster OLM v1 e2e** (optional): `make olmv1-e2e-test` after `./hack/olmv1/install.sh` (requires OLM v1-enabled OpenShift)

## Release enablement

See [RELEASE_ENABLEMENT.md](./RELEASE_ENABLEMENT.md).

## Further reading

- [OpenShift Cluster extensions](https://docs.redhat.com/en/documentation/openshift_container_platform/latest/html/extensions/cluster-extensions)
- [Install an extension (OLM)](https://operator-framework.github.io/operator-controller/tutorials/install-extension/)
- [OLM v1 limitations](https://operator-framework.github.io/operator-controller/project/olmv1_limitations/)
