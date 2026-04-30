#!/usr/bin/env bash
set -euo pipefail

if [[ $# -lt 1 ]]; then
  echo "usage: $0 <requestNo>" >&2
  exit 2
fi

REQUEST_NO="$1"
API="${API:-http://127.0.0.1:18000}"
TOKEN="${TOKEN:-$(cat /tmp/admin.token)}"
TIMEOUT_SECONDS="${TIMEOUT_SECONDS:-180}"
INTERVAL_SECONDS="${INTERVAL_SECONDS:-5}"

deadline=$(( $(date +%s) + TIMEOUT_SECONDS ))

while true; do
  RESP=$(curl -sf "${API}/api/v1/integrations/agent/deploy-requests/${REQUEST_NO}" \
    -H "Authorization: Bearer ${AGENT_TOKEN:-change-me-agent-token}")

  STATUS=$(RESP_JSON="${RESP}" python3 - <<'PY'
import json, os
payload = json.loads(os.environ["RESP_JSON"])
data = payload.get("data", {})
print(data.get("executionStatus", ""))
PY
)
  echo "executionStatus=${STATUS}"

  if [[ "${STATUS}" == "succeeded" ]]; then
    echo "PASS"
    exit 0
  fi
  if [[ "${STATUS}" == "failed" ]]; then
    echo "${RESP}"
    echo "FAIL: execution failed"
    exit 1
  fi

  if (( $(date +%s) >= deadline )); then
    echo "FAIL: timeout waiting for auto execution"
    exit 1
  fi

  curl -sf -X POST "${API}/api/v1/integrations/agent/deploy-requests/${REQUEST_NO}/sync-approval" \
    -H "Authorization: Bearer ${AGENT_TOKEN:-change-me-agent-token}" \
    -H "Content-Type: application/json" >/dev/null || true
  sleep "${INTERVAL_SECONDS}"
done
