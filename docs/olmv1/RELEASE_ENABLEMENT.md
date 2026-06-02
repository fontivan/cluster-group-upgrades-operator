# TALM OLM v1 release technical enablement

## Operator identity

| Item | Value |
|------|-------|
| Product name | Topology Aware Lifecycle Manager (TALM) |
| OLM package (FBC / Konflux) | `topology-aware-lifecycle-manager` |
| Legacy bundle package label | `cluster-group-upgrades-operator` (SDK metadata; catalog uses TALM name) |
| Install mode | **AllNamespaces** (only supported mode) |
| Operator namespace | `openshift-cluster-group-upgrades` |
| Konflux application | `topology-aware-lifecycle-manager-5-0` |
| FBC image (dev) | `quay.io/redhat-user-workloads/telco-5g-tenant/topology-aware-lifecycle-manager-fbc-5-0` |
| Red Hat operator image | `registry.redhat.io/openshift4/topology-aware-lifecycle-manager-rhel9-operator` |

## OLM v1 compatibility checklist

- [x] Bundle format `registry+v1`
- [x] `AllNamespaces` install mode supported; other modes disabled in CSV
- [x] No CSV `webhookdefinitions` in shipped bundle
- [x] No `olm.gvk.required` / `olm.package.required` / `olm.constraint` FBC properties
- [x] ClusterExtension + ClusterCatalog reference CRs in `deploy/olmv1/`
- [x] Automated bundle validation in `ci-job` (`make olmv1-bundle-validate`)
- [x] Install/verify scripts under `hack/olmv1/`
- [ ] Resource utilization table completed in [RESOURCE_UTILIZATION.md](./RESOURCE_UTILIZATION.md) (run `measure-resource-utilization.sh` on reference hub)
- [ ] Optional: openshift/release job for OLM v1 e2e (cluster profile with OLM v1 enabled)

## Documentation links (Done checklist)

| Deliverable | Location |
|-------------|----------|
| OLM v1 install guide | [docs/olmv1/README.md](./README.md) |
| ClusterExtension reference CRs | [deploy/olmv1/](../../deploy/olmv1/) |
| Resource utilization | [docs/olmv1/RESOURCE_UTILIZATION.md](./RESOURCE_UTILIZATION.md) |
| This enablement sheet | [docs/olmv1/RELEASE_ENABLEMENT.md](./RELEASE_ENABLEMENT.md) |
| OpenShift Cluster extensions | https://docs.redhat.com/en/documentation/openshift_container_platform/latest/html/extensions/cluster-extensions |
| OLM v1 install tutorial | https://operator-framework.github.io/operator-controller/tutorials/install-extension/ |

## RDS

Reference CRs for OLM v1 installation are maintained in this repository under `deploy/olmv1/`. Telco hub / ZTP **reference configuration** changes for OLM v1 are tracked separately (out of scope for CNF-23105).

## Verification commands

```bash
make olmv1-bundle-validate
go test ./tests/olmv1/...
# On OLM v1 cluster:
./hack/olmv1/install.sh && ./hack/olmv1/verify-install.sh
```
