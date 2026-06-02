#!/usr/bin/env bash
# Collect operator pod CPU/memory metrics for resource utilization comparison.
# Run twice on the same cluster: once with a namespace-scoped OLM (Classic) baseline
# (if available) and once after OLM v1 AllNamespaces installation, then compare outputs.
set -euo pipefail

TALM_NAMESPACE="${TALM_NAMESPACE:-openshift-cluster-group-upgrades}"
DEPLOYMENT_NAME="${DEPLOYMENT_NAME:-cluster-group-upgrades-controller-manager-v2}"
DURATION_SECONDS="${DURATION_SECONDS:-300}"
INTERVAL_SECONDS="${INTERVAL_SECONDS:-30}"
OUTPUT_DIR="${OUTPUT_DIR:-./artifacts/olmv1-resource-utilization/$(date -u +%Y%m%dT%H%M%SZ)}"
SCENARIO_LABEL="${SCENARIO_LABEL:-all-namespaces}"

mkdir -p "${OUTPUT_DIR}"

pod="$(oc get pods -n "${TALM_NAMESPACE}" -l control-plane=controller-manager -o jsonpath='{.items[0].metadata.name}')"
if [[ -z "${pod}" ]]; then
  echo "No controller-manager pod found in ${TALM_NAMESPACE}" >&2
  exit 1
fi

{
  echo "scenario=${SCENARIO_LABEL}"
  echo "namespace=${TALM_NAMESPACE}"
  echo "pod=${pod}"
  echo "duration_seconds=${DURATION_SECONDS}"
  echo "interval_seconds=${INTERVAL_SECONDS}"
  echo "timestamp_utc=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
} > "${OUTPUT_DIR}/metadata.txt"

oc get deployment "${DEPLOYMENT_NAME}" -n "${TALM_NAMESPACE}" -o yaml > "${OUTPUT_DIR}/deployment.yaml"
oc top pod "${pod}" -n "${TALM_NAMESPACE}" --containers > "${OUTPUT_DIR}/top-once.txt" 2>&1 || true

samples="${OUTPUT_DIR}/samples.csv"
echo "elapsed_seconds,cpu_millicores,memory_mi" > "${samples}"

end=$((SECONDS + DURATION_SECONDS))
while [[ "${SECONDS}" -lt "${end}" ]]; do
  elapsed=$((DURATION_SECONDS - (end - SECONDS)))
  line="$(oc top pod "${pod}" -n "${TALM_NAMESPACE}" --no-headers 2>/dev/null | awk '{print $2","$3}')"
  if [[ -n "${line}" ]]; then
    echo "${elapsed},${line}" >> "${samples}"
  fi
  sleep "${INTERVAL_SECONDS}"
done

if command -v jq >/dev/null 2>&1 && [[ -s "${samples}" ]]; then
  jq -Rn '
    [inputs | select(length>0) | split(",") | {elapsed: (.[0]|tonumber), cpu: (.[1]|gsub("m";"")|tonumber), mem: (.[2]|gsub("Mi";"")|tonumber)}]
    | select(length>0)
    | {
        sample_count: length,
        cpu_millicores: {min: (map(.cpu)|min), max: (map(.cpu)|max), avg: ((map(.cpu)|add)/length)},
        memory_mi: {min: (map(.mem)|min), max: (map(.mem)|max), avg: ((map(.mem)|add)/length)}
      }
  ' <(tail -n +2 "${samples}") > "${OUTPUT_DIR}/summary.json"
fi

echo "Resource utilization samples written to ${OUTPUT_DIR}"
