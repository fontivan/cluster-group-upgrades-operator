# TALM resource utilization: install mode comparison

## Context

CNF-23105 requires documenting CPU and memory impact when the operator runs under OLM v1 **AllNamespaces** install mode versus a namespace-restricted (OwnNamespace / SingleNamespace) deployment.

The current TALM bundle declares **only** `AllNamespaces` as supported. The controller manager already uses cluster-scoped RBAC and a default cluster-wide informer cache (with a label filter on `ManifestWork` only). For this operator, moving from a legacy OwnNamespace OLM (Classic) subscription to OLM v1 AllNamespaces is therefore expected to show **small additional overhead** at idle, with cost scaling mainly with the number of watched API objects—not with the install mode flag itself.

## Measurement procedure

1. Use a hub cluster with ACM CRDs and representative policy/CGU load (or the integration test fixture).
2. Record baseline (if a classic OwnNamespace install is still available in your environment):
   ```bash
   SCENARIO_LABEL=own-namespace TALM_NAMESPACE=<legacy-ns> ./hack/olmv1/measure-resource-utilization.sh
   ```
3. Install via OLM v1 reference CRs and verify:
   ```bash
   ./hack/olmv1/install.sh
   ./hack/olmv1/verify-install.sh
   SCENARIO_LABEL=all-namespaces ./hack/olmv1/measure-resource-utilization.sh
   ```
4. Compare `summary.json` under `artifacts/olmv1-resource-utilization/<timestamp>/` for each scenario.

Default sampling: 300 seconds, 30 second interval (`DURATION_SECONDS`, `INTERVAL_SECONDS`).

## Results

Fill in after running on a reference hub cluster (4.16+ with OLM v1 TechPreview/GA per stream):

| Scenario | CPU (avg / max mCPU) | Memory (avg / max Mi) | Notes |
|----------|----------------------|------------------------|-------|
| OwnNamespace (OLM Classic baseline) | _TBD_ | _TBD_ | Optional; only if legacy install still used |
| AllNamespaces (OLM v1 ClusterExtension) | _TBD_ | _TBD_ | Primary acceptance path |

**JIRA / epic link:** attach the artifact directories or paste `summary.json` contents when closing CNF-23105.

## Expected outcome

- Idle operator pod remains near CSV requests (`100m` CPU, `20Mi` memory) with short spikes during reconciliation.
- AllNamespaces should not materially increase utilization versus OwnNamespace for TALM because both paths already reconcile cluster-scoped and multi-namespace ACM resources.
