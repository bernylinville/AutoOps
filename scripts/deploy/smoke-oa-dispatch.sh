#!/usr/bin/env bash
set -euo pipefail

API="${API:-http://127.0.0.1:18000}"
TOKEN="${TOKEN:-$(cat /tmp/admin.token)}"
CLUSTER_TARGET_ID="${CLUSTER_TARGET_ID:-1}"
APPROVER_ADMIN_ID="${APPROVER_ADMIN_ID:-89}"
IMAGE="${IMAGE:-pukka-all-images-cn-shanghai.cr.volces.com/proxy/nginx:1.27.4-alpine}"
RELEASE_NAME="${RELEASE_NAME:-oa-smoke-$(date +%s)}"
NAMESPACE="${NAMESPACE:-ao-direct-${RELEASE_NAME}}"

REQ_BODY=$(cat <<JSON
{
  "clusterTargetId": ${CLUSTER_TARGET_ID},
  "mode": "direct",
  "resourceType": "deployment",
  "releaseName": "${RELEASE_NAME}",
  "namespace": "${NAMESPACE}",
  "image": "${IMAGE}",
  "replicas": 1,
  "serviceEnabled": true,
  "serviceType": "ClusterIP",
  "servicePort": 80,
  "targetPort": 80,
  "ttlHours": 1,
  "approverAdminId": ${APPROVER_ADMIN_ID},
  "reason": "OA smoke test"
}
JSON
)

echo "==> Creating deploy request..."
RESP=$(curl -sf -X POST "${API}/api/v1/deploy/requests" \
  -H "Authorization: Bearer ${TOKEN}" \
  -H "Content-Type: application/json" \
  -d "${REQ_BODY}")
echo "${RESP}"

RESP_JSON="${RESP}" python3 - <<'PY'
import json, sys
import os
payload = json.loads(os.environ["RESP_JSON"])
data = payload.get("data", {})
print(f"requestNo={data.get('requestNo')} dispatch={data.get('approvalDispatchStatus')} procId={data.get('dingtalkProcessInstanceId')}")
if data.get("approvalDispatchStatus") != "dispatched":
    raise SystemExit("FAIL: approvalDispatchStatus != dispatched")
if not data.get("dingtalkProcessInstanceId"):
    raise SystemExit("FAIL: dingtalkProcessInstanceId empty")
print("PASS")
PY
