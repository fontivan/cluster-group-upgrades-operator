#!/usr/bin/env bash
# Post-install checks for TALM installed via OLM v1 / AllNamespaces.
set -euo pipefail

TALM_NAMESPACE="${TALM_NAMESPACE:-openshift-cluster-group-upgrades}"
PACKAGE_NAME="${PACKAGE_NAME:-topology-aware-lifecycle-manager}"
DEPLOYMENT_NAME="${DEPLOYMENT_NAME:-cluster-group-upgrades-controller-manager-v2}"

echo "Verifying ClusterExtension status..."
installed="$(oc get clusterextension "${PACKAGE_NAME}" -o jsonpath='{.status.conditions[?(@.type=="Installed")].status}')"
if [[ "${installed}" != "True" ]]; then
  echo "ClusterExtension ${PACKAGE_NAME} is not Installed" >&2
  oc describe clusterextension "${PACKAGE_NAME}" || true
  exit 1
fi

echo "Verifying controller deployment..."
oc wait --for=condition=Available "deployment/${DEPLOYMENT_NAME}" -n "${TALM_NAMESPACE}" --timeout=10m

echo "Verifying CRDs are established..."
for crd in clustergroupupgrades.ran.openshift.io precachingconfigs.ran.openshift.io imagebasedgroupupgrades.lcm.openshift.io; do
  oc wait --for=condition=Established "crd/${crd}" --timeout=5m
done

echo "Verifying AllNamespaces install mode in installed CSV..."
csv="$(oc get csv -n "${TALM_NAMESPACE}" -o json | jq -r --arg pkg "${PACKAGE_NAME}" '.items[] | select(.spec.customresourcedefinitions!=null) | select(.metadata.name|startswith($pkg)) | .metadata.name' | head -1)"
if [[ -z "${csv}" ]]; then
  csv="$(oc get csv -n "${TALM_NAMESPACE}" -o name | head -1 | cut -d/ -f2)"
fi
if [[ -n "${csv}" ]]; then
  if ! oc get "csv/${csv}" -n "${TALM_NAMESPACE}" -o json | jq -e '.spec.installModes[] | select(.type=="AllNamespaces" and .supported==true)' >/dev/null; then
    echo "Installed CSV does not report AllNamespaces support" >&2
    exit 1
  fi
fi

echo "OLM v1 install verification passed."
