#!/usr/bin/env python3
import json
import os
import sys

import requests


def main() -> int:
    api = os.environ.get("AUTOOPS_API", "http://127.0.0.1:18000")
    token = os.environ.get("AGENT_TOKEN")
    requester_external_id = os.environ.get("REQUESTER_EXTERNAL_ID", "smoke-dingtalk-admin")
    requester_display_name = os.environ.get("REQUESTER_DISPLAY_NAME", "Hermes Smoke")
    if not token:
        print("AGENT_TOKEN is required", file=sys.stderr)
        return 2

    body = {
        "clusterTargetId": 1,
        "mode": "direct",
        "resourceType": "deployment",
        "releaseName": f"hermes-smoke-{os.getpid()}",
        "namespace": f"ao-direct-hermes-smoke-{os.getpid()}",
        "image": "pukka-all-images-cn-shanghai.cr.volces.com/proxy/nginx:1.27.4-alpine",
        "replicas": 1,
        "serviceEnabled": True,
        "serviceType": "ClusterIP",
        "servicePort": 80,
        "targetPort": 80,
        "ttlHours": 1,
        "approverAdminId": 89,
        "reason": "hermes contract test",
        "requesterExternalType": "dingtalk",
        "requesterExternalId": requester_external_id,
        "requesterDisplayName": requester_display_name,
        "chatContext": {
            "provider": "dingtalk",
            "chat_id": "smoke_chat",
            "at_mobiles": [],
            "at_user_ids": [],
            "sender_external_id": requester_external_id,
            "origin_message_id": "msg_smoke"
        }
    }

    headers = {"Authorization": f"Bearer {token}"}
    create_resp = requests.post(
        f"{api}/api/v1/integrations/agent/deploy-requests",
        json=body,
        headers=headers,
        timeout=15,
    )
    create_resp.raise_for_status()
    create_payload = create_resp.json()
    if "data" not in create_payload or "requestNo" not in create_payload["data"]:
        print(json.dumps(create_payload, ensure_ascii=False, indent=2))
        raise KeyError("requestNo")
    request_no = create_payload["data"]["requestNo"]
    print(f"created requestNo={request_no}")

    status_resp = requests.get(
        f"{api}/api/v1/integrations/agent/deploy-requests/{request_no}/status",
        headers=headers,
        timeout=15,
    )
    status_resp.raise_for_status()
    print(json.dumps(status_resp.json(), ensure_ascii=False, indent=2))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
