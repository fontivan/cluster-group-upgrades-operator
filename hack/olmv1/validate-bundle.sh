#!/usr/bin/env bash
# Validates the operator bundle meets OLM v1 installation requirements.
set -euo pipefail

PROJECT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
BUNDLE_DIR="${PROJECT_DIR}/bundle"
CSV_FILE="${BUNDLE_DIR}/manifests/cluster-group-upgrades-operator.clusterserviceversion.yaml"
ANNOTATIONS_FILE="${BUNDLE_DIR}/metadata/annotations.yaml"

if [[ ! -f "${CSV_FILE}" ]]; then
  echo "CSV not found at ${CSV_FILE}; run 'make bundle' first" >&2
  exit 1
fi

if ! command -v yq >/dev/null 2>&1; then
  if [[ -x "${PROJECT_DIR}/bin/yq" ]]; then
    YQ="${PROJECT_DIR}/bin/yq"
  else
    echo "yq is required (install via 'make yq' or PATH)" >&2
    exit 1
  fi
else
  YQ=yq
fi

echo "Checking bundle media type..."
if ! grep -q 'operators.operatorframework.io.bundle.mediatype.v1: registry+v1' "${ANNOTATIONS_FILE}"; then
  echo "bundle must use registry+v1 format" >&2
  exit 1
fi

echo "Checking install modes (AllNamespaces required, others unsupported)..."
install_modes="$("${YQ}" eval '.spec.installModes[] | .type + "=" + (.supported | tostring)' "${CSV_FILE}")"
if ! grep -q '^AllNamespaces=true$' <<<"${install_modes}"; then
  echo "AllNamespaces install mode must be supported" >&2
  exit 1
fi
for mode in OwnNamespace SingleNamespace MultiNamespace; do
  if grep -q "^${mode}=true$" <<<"${install_modes}"; then
    echo "${mode} must not be supported for OLM v1 AllNamespaces-only TALM" >&2
    exit 1
  fi
done

echo "Checking webhooks are not declared in the CSV..."
if [[ "$("${YQ}" eval '.spec.webhookdefinitions // ""' "${CSV_FILE}")" != "" ]]; then
  echo "OLM v1 compatibility test expects no CSV webhookdefinitions" >&2
  exit 1
fi

echo "Checking file-based catalog dependency properties are absent..."
if grep -rE 'olm\.(gvk|package|constraint)\.required' "${BUNDLE_DIR}" >/dev/null 2>&1; then
  echo "bundle must not declare olm.gvk/package/constraint.required properties" >&2
  exit 1
fi

echo "OLM v1 bundle validation passed."
