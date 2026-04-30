#!/usr/bin/env bash
set -euo pipefail

if [[ $# -lt 1 ]]; then
  echo "usage: $0 <requestId>" >&2
  exit 2
fi

REQUEST_ID="$1"
API="${API:-http://127.0.0.1:18000}"
TOKEN="${TOKEN:-$(cat /tmp/admin.token)}"

RESP=$(curl -sf "${API}/api/v1/deploy/requests/${REQUEST_ID}/notifications" \
  -H "Authorization: Bearer ${TOKEN}")
echo "${RESP}"

RESP_JSON="${RESP}" python3 - <<'PY'
import json
import os

payload = json.loads(os.environ["RESP_JSON"])
items = payload.get("data", [])
if not items:
    raise SystemExit("FAIL: no notification records")
if not any(item.get("status") == "sent" for item in items):
    raise SystemExit("FAIL: no sent notification")
print("PASS")
PY
