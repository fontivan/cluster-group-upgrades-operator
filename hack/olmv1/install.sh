#!/usr/bin/env bash
# Install TALM on an OLM v1-enabled OpenShift cluster using reference CRs.
set -euo pipefail

PROJECT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
TALM_NAMESPACE="${TALM_NAMESPACE:-openshift-cluster-group-upgrades}"
CATALOG_NAME="${CATALOG_NAME:-topology-aware-lifecycle-manager-catalog}"
PACKAGE_NAME="${PACKAGE_NAME:-topology-aware-lifecycle-manager}"
FBC_IMAGE="${FBC_IMAGE:-quay.io/redhat-user-workloads/telco-5g-tenant/topology-aware-lifecycle-manager-fbc-5-0:latest}"
CLUSTER_EXTENSION_VERSION="${CLUSTER_EXTENSION_VERSION:-5.0.0}"
CLUSTER_EXTENSION_CHANNEL="${CLUSTER_EXTENSION_CHANNEL:-stable}"
SKIP_IDMS="${SKIP_IDMS:-false}"

if ! oc get clusterextension >/dev/null 2>&1; then
  echo "ClusterExtension API not found. Enable OLM v1 on the cluster first." >&2
  exit 1
fi

if [[ "${SKIP_IDMS}" != "true" ]]; then
  echo "Applying ImageDigestMirrorSet..."
  oc apply -f "${PROJECT_DIR}/deploy/olmv1/imagedigestmirrorset.yaml"
fi

echo "Applying namespace and installer ServiceAccount..."
oc apply -f "${PROJECT_DIR}/deploy/olmv1/namespace.yaml"
oc apply -f "${PROJECT_DIR}/deploy/olmv1/serviceaccount.yaml"
oc apply -f "${PROJECT_DIR}/deploy/olmv1/installer-clusterrolebinding.yaml"

echo "Applying ClusterCatalog (${CATALOG_NAME})..."
cat <<EOF | oc apply -f -
apiVersion: olm.operatorframework.io/v1
kind: ClusterCatalog
metadata:
  name: ${CATALOG_NAME}
spec:
  source:
    type: Image
    image:
      ref: ${FBC_IMAGE}
      pollIntervalMinutes: 10
EOF

echo "Waiting for ClusterCatalog to serve..."
for i in $(seq 1 30); do
  status="$(oc get clustercatalog "${CATALOG_NAME}" -o jsonpath='{.status.conditions[?(@.type=="Serving")].status}' 2>/dev/null || true)"
  if [[ "${status}" == "True" ]]; then
    break
  fi
  sleep 10
done

echo "Applying ClusterExtension..."
cat <<EOF | oc apply -f -
apiVersion: olm.operatorframework.io/v1
kind: ClusterExtension
metadata:
  name: ${PACKAGE_NAME}
spec:
  namespace: ${TALM_NAMESPACE}
  serviceAccount:
    name: topology-aware-lifecycle-manager-installer
  source:
    sourceType: Catalog
    catalog:
      name: ${CATALOG_NAME}
      packageName: ${PACKAGE_NAME}
      channels:
      - ${CLUSTER_EXTENSION_CHANNEL}
      version: "${CLUSTER_EXTENSION_VERSION}"
EOF

echo "Waiting for ClusterExtension installation..."
for i in $(seq 1 30); do
  status="$(oc get clusterextension "${PACKAGE_NAME}" -o jsonpath='{.status.conditions[?(@.type=="Installed")].status}' 2>/dev/null || true)"
  if [[ "${status}" == "True" ]]; then
    echo "ClusterExtension installed."
    exit 0
  fi
  sleep 10
done

echo "ClusterExtension did not reach Installed=True in time." >&2
oc describe clusterextension "${PACKAGE_NAME}" || true
exit 1
