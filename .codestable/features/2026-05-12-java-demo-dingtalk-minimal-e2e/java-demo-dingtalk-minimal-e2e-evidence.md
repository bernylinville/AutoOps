# java-demo DingTalk minimal E2E evidence

- Feature: `2026-05-12-java-demo-dingtalk-minimal-e2e`
- Evidence started: `2026-05-12T14:13:29+08:00`
- Scope: DingTalk 群用户原话 → opsclaw Hermes → AutoOps Agent API → DingTalk OA → Jenkins → Harbor → AutoOps Direct Kubernetes → NodePort → deploy_bot 回群。
- Secret policy: 本文件只记录资源名、状态、脱敏配置和可公开内网访问证据；不记录 token、密码、PAT、webhook、kubeconfig 或 Secret value。

## 1. Preflight evidence

### 1.1 AutoOps and Kubernetes health

- AutoOps health endpoint: `GET http://10.0.17.206/api/v1/healthz` returned `{"status":"ok"}`.
- Kubernetes context: `kubernetes-admin@cluster.local`.
- Nodes ready:
  - `pukka-k8s-cp01` / `10.0.17.43`
  - `pukka-k8s-cp02` / `10.0.17.44`
  - `pukka-k8s-cp03` / `10.0.17.45`
  - `pukka-k8s-wk01` / `10.0.17.40`
  - `pukka-k8s-wk02` / `10.0.17.41`
  - `pukka-k8s-wk03` / `10.0.17.42`
- AutoOps namespace pods observed running:
  - `autoops-api-6dbb9c58fc-gfqx4` `1/1 Running`
  - `autoops-postgres-6f8fc7cb85-nbnk4` `1/1 Running`
  - `autoops-valkey-6896587fdd-4shlr` `1/1 Running`
  - `autoops-web-7997bd8787-89dmg` `1/1 Running`
- AutoOps API selected env: `AUTOOPS_SKIP_PIPELINE_SCAN=true`; `JWT_SECRET=<set>`.

### 1.2 AutoOps runtime configuration and Secret key presence

`autoops-runtime` Secret exists. Key presence only, values not read:

- `AGENT_BEARER_TOKEN`
- `DEPLOY_BOT_SECRET`
- `DEPLOY_BOT_WEBHOOK_URL`
- `DINGTALK_APPROVAL_CLIENT_ID`
- `DINGTALK_APPROVAL_CLIENT_SECRET`
- `DINGTALK_APPROVAL_ORIGINATOR_DEPT_ID`
- `DINGTALK_APPROVAL_PROCESS_CODE`
- plus unrelated runtime keys such as `JWT_SECRET`, database/cache passwords, FlashDuty and heartbeat keys.

`autoops-api-runtime` ConfigMap template confirms the intended non-secret project-onboarding contract:

- `project_onboarding.enabled: true`
- `allowed_git_hosts: ["gayhub.seeingtv.com"]`
- `shared_jenkins_job_name: "java-demo-build"`
- `default_jenkins_server_id: 1`
- `default_harbor_server_id: 2`
- `default_harbor_project: "java-demo"`
- `default_approver_admin_id: 89`
- `test_cluster_target_id: 2`
- `namespace_prefix: "ao-direct"`
- `default_service_port: 80`
- `default_target_port: 8080`
- `nodeport_access_host: "10.0.17.206"` is rendered in config template, but the design already treats Go consumption of this field as a later roadmap item.
- `deploy_bot.provider: "dingtalk"`; `deploy_bot.enabled` defaults from runtime env with a template default of `true`; webhook and secret are redacted.

### 1.3 AutoOps database preflight

Cluster targets:

```text
1|pukka-devtest-k8s|devtest|direct_enabled=true|git_ops_enabled=true|has_kubeconfig_ref=true|approver=89|namespace_prefix=ao-direct
2|pukka-staging-k8s|staging|direct_enabled=true|git_ops_enabled=true|has_kubeconfig_ref=true|approver=89|namespace_prefix=ao-direct
```

Jenkins / Harbor accounts:

```text
1|admin|type=4|jenkins.jenkins.svc.cluster.local|8080
2|admin|type=5|10.0.17.205|80
```

`java-demo` application and deploy profiles:

```text
4|java-demo|git@gayhub.seeingtv.com:ipaas/java-demo.git
2|app=4|code=java-demo|env=devtest|enabled=true|clusterTarget=1|namespace=java-demo-devtest|release=java-demo-devtest|service=NodePort 80->8080|jenkins=1/java-demo-build|harbor=2/java-demo/java-demo|gitRef=main|approver=89
3|app=4|code=java-demo|env=staging|enabled=true|clusterTarget=2|namespace=java-demo-staging|release=java-demo-staging|service=ClusterIP 80->8080|jenkins=1/java-demo-build|harbor=2/java-demo/java-demo|gitRef=main|approver=89
4|app=4|code=java-demo|env=test|enabled=true|clusterTarget=2|namespace=ao-direct-java-demo-test|release=java-demo|service=NodePort 80->8080|jenkins=1/java-demo-build|harbor=2/java-demo/java-demo|gitRef=main|approver=89
```

Recent successful request proving the existing Direct NodePort path can finish:

```text
DR20260509170924.086960785|request=succeeded|approval=approved|execution=succeeded|pipeline=succeeded|stage=deploy|namespace=ao-direct-java-demo-test|release=java-demo|service=NodePort|image=10.0.17.205:80/java-demo/java-demo:main-20260509-171123-c5bdf39
```

### 1.4 Existing Kubernetes and NodePort access evidence

Current test namespace resources:

```text
deployment.apps/java-demo  READY 1/1  image=10.0.17.205:80/java-demo/java-demo:main-20260509-171123-c5bdf39
service/java-demo          TYPE NodePort  PORT(S) 80:32580/TCP
pod/java-demo-8564d7d7f4-8p2qx  READY 1/1  STATUS Running
```

Access check:

```text
curl http://10.0.17.40:32580/
{"status":"ok","message":"AutoOps Java demo is running","application":"java-demo","profile":"test",...}
```

### 1.5 Jenkins, Harbor, java-demo repo and cluster source evidence

- Jenkins endpoint `http://10.0.17.204/` returned HTTP `403` without credentials, which is expected for unauthenticated access.
- Harbor endpoint `http://10.0.17.205/` returned HTTP `200`.
- Local `/home/kchou/Code/java-demo` is on `main`, remote `git@gayhub.seeingtv.com:ipaas/java-demo.git`.
- `/home/kchou/Code/java-demo/Jenkinsfile` already emits current AutoOps parser-compatible output:
  - `IMAGE_TAG=${env.IMAGE_TAG}`
  - `镜像地址=${env.FULL_IMAGE}`
  - buildah push lines push `FULL_IMAGE`.
- Kubespray inventory source: `/home/kchou/Code/kubespray/inventory/pukka-cluster/inventory.ini` lists `pukka-k8s-cp01..03` and `pukka-k8s-wk01..03` with the IPs above.
- pukka-gitops AutoOps values source: `/home/kchou/Code/pukka-gitops/apps/autoops/values.yaml`; rendered runtime ConfigMap evidence is recorded in section 1.2.

### 1.6 Hermes preflight on opsclaw

- opsclaw reachable by SSH.
- Hermes gateway process observed:

```text
/home/pukka/.hermes/hermes-agent/venv/bin/python -m hermes_cli.main gateway run --replace
```

- Skill routing evidence from `~/.hermes/skills/devops/deploy-via-autoops/SKILL.md` and `~/.hermes/hermes-agent/skills/devops/deploy-via-autoops/SKILL.md`:
  - Git URL messages must call `project-onboard-build-deploy`.
  - The exact user prompt is listed as an example.
  - `需要能对外访问` maps to `exposureMode=nodeport`.
  - The skill forbids asking users for `namespace`, `image`, `clusterTarget`, `Jenkins job`, `Harbor project/repository`.

### 1.7 Preflight conclusion

Preflight is **triggerable with one tactical blocker fixed in this feature**:

- Triggerable: AutoOps health, cluster, profile, Jenkins account, Harbor account, DingTalk OA keys, deploy_bot keys, java-demo Profile, existing NodePort deployment and Jenkinsfile parser-compatible output are present.
- Tactical blocker found: opsclaw Hermes script and skill template used legacy `chatId` / `atUserIds` inside `chatContext`; AutoOps notifier contract uses snake_case `chat_id` / `at_user_ids`.
- Out-of-scope observations kept for later roadmap items:
  - `nodeport_access_host` is rendered but not treated here as a completion blocker.
  - Generic `AUTOOPS_IMAGE` parser contract is not implemented here because the current Jenkinsfile already emits `IMAGE_TAG` and image address.

## 2. Tactical fix evidence

### 2.1 Changed remote files on opsclaw

Remote files modified on `opsclaw`:

- `~/.hermes/scripts/autoops-deploy.sh`
- `~/.hermes/skills/devops/deploy-via-autoops/SKILL.md`
- `~/.hermes/hermes-agent/skills/devops/deploy-via-autoops/SKILL.md`

Backup copies were created on opsclaw with timestamp suffixes before editing. No AutoOps Go code or `java-demo` Jenkinsfile was modified for this feature.

### 2.2 Exact tactical change

Only `chatContext` key spelling was changed from legacy camelCase to snake_case:

```text
"chatId"     -> "chat_id"
"atUserIds"  -> "at_user_ids"
```

The script now contains snake_case keys in all request modes:

```text
~/.hermes/scripts/autoops-deploy.sh:59  "chat_id": chat_id,
~/.hermes/scripts/autoops-deploy.sh:60  "at_user_ids": [sender_id],
~/.hermes/scripts/autoops-deploy.sh:77  "chat_id": chat_id,
~/.hermes/scripts/autoops-deploy.sh:78  "at_user_ids": [sender_id],
~/.hermes/scripts/autoops-deploy.sh:109 "chat_id": chat_id,
~/.hermes/scripts/autoops-deploy.sh:110 "at_user_ids": [sender_id],
```

Both skill templates now show:

```json
"chat_id": "<chat id>",
"at_user_ids": ["<sender DingTalk userId>"]
```

### 2.3 Validation

- `bash -n ~/.hermes/scripts/autoops-deploy.sh` passed on opsclaw.
- Dry-run used a temporary fake `curl` on opsclaw so no real AutoOps request was created.
- Dry-run command shape:

```bash
bash ~/.hermes/scripts/autoops-deploy.sh project-onboard-build-deploy \
  git@gayhub.seeingtv.com:ipaas/java-demo.git test main nodeport \
  '帮我把 git@gayhub.seeingtv.com:ipaas/java-demo.git 的 main 分支重新部署到测试环境，需要能对外访问'
```

Dry-run captured request subset:

```json
{
  "endpoint": "/api/v1/integrations/agent/project-onboard-build-deploy",
  "gitRepoUrl": "git@gayhub.seeingtv.com:ipaas/java-demo.git",
  "env": "test",
  "gitRef": "main",
  "exposureMode": "nodeport",
  "chatContextKeys": [
    "at_user_ids",
    "chat_id",
    "externalAccessRequired",
    "intent",
    "provider",
    "source"
  ],
  "legacyCamelCasePresent": false
}
```

Step 2 exit signal is satisfied: Hermes can construct the expected Agent request, and the current java-demo Jenkinsfile already emits `IMAGE_TAG` / image address that the current AutoOps parser can consume.

## 3. Current stop point

Implementation is paused at checklist step 3 because the approved design requires **real DingTalk group entry**. A naked `curl` or dry-run would violate the acceptance contract.

Required external action for the next step:

```text
在钉钉群里对 opsclaw Hermes agent 发送：
帮我把 git@gayhub.seeingtv.com:ipaas/java-demo.git 的 main 分支重新部署到测试环境，需要能对外访问
```

Expected step 3 evidence to append after the group trigger:

- group or Hermes response includes `requestNo`;
- AutoOps status for that request shows `workflowKind=build_deploy`, `mode=direct`, `approvalStatus=pending`;
- request body semantics match `gitRepoUrl`, `env=test`, `gitRef=main`, `exposureMode=nodeport`, snake_case `chatContext`.

## 4. Real DingTalk trigger tracking — `DR20260512163231.431627832`

Tracking time: `2026-05-12T17:44+08:00`.

### 4.1 Request and approval state

User-provided group notification confirms the real DingTalk entry produced request `DR20260512163231.431627832`, and DingTalk OA was approved.

AutoOps database state for this request:

```text
request_no=DR20260512163231.431627832
source=agent
workflowKind=build_deploy
mode=direct
namespace=ao-direct-java-demo-test
release=java-demo
serviceType=NodePort
requestStatus=failed
approvalStatus=approved
executionStatus=failed
pipelineStatus=failed
currentPipelineStage=build
finishedAt=2026-05-12 17:06:59+08
pipelineError=Jenkins 构建失败，结果=FAILURE
approvalDispatchStatus=dispatched
approvalDispatchMessage=钉钉审批实例状态: status=COMPLETED result=agree
approvedAt=2026-05-12 16:33:01+08
```

No manual approval bypass was observed; the request reached `approvalStatus=approved` before build failure was recorded.

Two duplicate real DingTalk submissions remain pending approval and were not used for this evidence trail:

```text
DR20260512163248.423566394|pending_approval|pending|pending|pending
DR20260512163337.185849092|pending_approval|pending|pending|pending
```

### 4.2 Pipeline and Jenkins evidence

Pipeline run evidence:

```text
pipelineRunID=21
requestID=21
status=failed
currentStage=build
jenkinsServerID=1
jenkinsJob=java-demo-build
gitRef=main
jenkinsQueueID=145
jenkinsBuildNumber=69 (recorded in build stage detail)
jenkinsBuildURL=http://10.0.17.204/job/java-demo-build/69/
harborServerID=2
harborProject=java-demo
harborRepository=java-demo
lastError=Jenkins 构建失败，结果=FAILURE
startedAt=2026-05-12 16:33:33+08
finishedAt=2026-05-12 17:06:59+08
```

Build stage detail:

```text
stage=build
status=failed
Jenkins result=FAILURE
Jenkins build=69
Jenkins URL=http://10.0.17.204/job/java-demo-build/69/
Jenkins duration=1982886 ms
parameters: APPLICATION_CODE=java-demo, GIT_REF=main, HARBOR_PROJECT=java-demo, HARBOR_REGISTRY=10.0.17.205, HARBOR_REPOSITORY=java-demo, IMAGE_TAG=<empty, auto-generated>
errorMessage=Jenkins 构建失败，结果=FAILURE
```

Sanitized Jenkins console root cause:

```text
Maven build succeeded.
Buildah image build succeeded.
Generated image tag: main-20260512-164003-c5bdf39
Generated image ref used by Jenkins: 10.0.17.205/java-demo/java-demo:main-20260512-164003-c5bdf39
Push stage failed while uploading blob:
received unexpected HTTP status: 504 Gateway Timeout
Jenkins final error: script returned exit code 125
Jenkins result: FAILURE
```

Failure classification: **Harbor push path timeout during Jenkins `buildah push`**. This is after source checkout and image build, before AutoOps can parse a successful image and before Direct Kubernetes deploy.

### 4.3 Harbor and Kubernetes impact

Because Jenkins failed in the push stage, AutoOps did not record `artifactTag`, `artifactDigest`, `plannedImageRef`, or `finalImageRef` for this request.

Unauthenticated Docker Registry manifest checks for the failed tag returned `401`, so they do not prove artifact presence or absence:

```text
http://10.0.17.205/v2/java-demo/java-demo/manifests/main-20260512-164003-c5bdf39 -> 401
http://10.0.17.205:80/v2/java-demo/java-demo/manifests/main-20260512-164003-c5bdf39 -> 401
```

Current Kubernetes test namespace remains on the previous successful image, not the failed build tag:

```text
deployment/java-demo ready=1/1 image=10.0.17.205:80/java-demo/java-demo:main-20260509-171123-c5bdf39
service/java-demo type=NodePort port=80 targetPort=8080 nodePort=32580
pod/java-demo-8564d7d7f4-8p2qx Running
```

Current NodePort still serves the old deployed version:

```text
curl http://10.0.17.40:32580/
{"status":"ok","message":"AutoOps Java demo is running","application":"java-demo","profile":"test",...}
```

### 4.4 Notification evidence

AutoOps notification record:

```text
channel=dingtalk_robot
stage=failed
status=sent
sentAt=2026-05-12 17:07:05+08
```

The user also pasted the group-visible deploy_bot failure message:

```text
AutoOps 部署结果 - DR20260512163231.431627832
状态: failed
错误信息: Jenkins 构建失败，结果=FAILURE
完成时间: 2026-05-12T17:06:59+08:00
```

### 4.5 Stop condition and next action

The E2E currently stops at checklist step 4 with an explicit stage error. Steps 5-6 success path cannot be completed for request `DR20260512163231.431627832` because no new image was pushed successfully and no Direct deploy stage ran.

Recommended immediate follow-up outside this feature's minimal tracking scope:

1. Treat this as a Jenkins/Harbor push-path incident: Harbor returned `504 Gateway Timeout` during blob upload from Jenkins build pod.
2. Retry only after deciding whether to use one of the duplicate pending requests or create a fresh request; current request `DR20260512163231.431627832` is terminal `failed` and AutoOps has no dedicated retry/reset endpoint in the observed API surface.
3. If retrying repeats the same error, route to the later roadmap work item for Jenkins/Harbor build contract or an incident-recovery issue; do not hide it in this minimal E2E feature.

## 5. Incident recovery — Jenkins → Harbor push 504

Recovery time: `2026-05-13T13:13-13:39+08:00`.

### 5.1 Root cause evidence

The failed request `DR20260512163231.431627832` failed in Jenkins build `#69` while pushing blobs to Harbor through `10.0.17.205`.

Correlated evidence:

- Jenkins build log: Maven build and image build succeeded; `buildah push` failed with `received unexpected HTTP status: 504 Gateway Timeout` and exit code `125`.
- Harbor registry log: backend accepted `/v2/java-demo/java-demo/blobs/uploads/` requests and some POST requests took about `13-19s` before returning `202`.
- Envoy Gateway log: Harbor `/v2/` traffic was routed through `harbor-httproute` rule 0 to `harbor-core`.
- Harbor registry PVC had sufficient free space: about `478Gi` available, `5%` used.
- Harbor pods were running and low resource usage; no broad Harbor outage was observed.

Conclusion: the 504 was caused by the Gateway request/backend timeout being too short for slow Harbor registry upload operations backed by NFS storage, not by application build failure or disk exhaustion.

### 5.2 Fix applied

GitOps repo: `/home/kchou/Code/pukka-gitops`.

Changed file:

```text
platform/harbor/harbor-httproute.yaml
```

Persisted commit:

```text
03ff8ef Keep Harbor image uploads from timing out at the gateway
```

Live and GitOps desired state now set a longer timeout on the Harbor API/registry route:

```yaml
rules:
  - timeouts:
      request: 10m
      backendRequest: 10m
    matches:
      - path: { type: PathPrefix, value: /api/ }
      - path: { type: PathPrefix, value: /service/ }
      - path: { type: PathPrefix, value: /v2/ }
      - path: { type: PathPrefix, value: /chartrepo/ }
      - path: { type: PathPrefix, value: /c/ }
```

Validation before apply:

```text
kubectl apply --dry-run=server -f platform/harbor/harbor-httproute.yaml -> configured (server dry run)
```

ArgoCD after push:

```text
platform-harbor Synced Healthy revision=03ff8ef5942a3a88b601291ee01fa86fc7d95c1e
HTTPRoute spec.rules[0].timeouts request=10m backendRequest=10m
```

### 5.3 Verification build

A direct Jenkins verification build was triggered to keep the test focused on the repaired push path:

```text
job=java-demo-build
build=70
IMAGE_TAG=timeoutfix-20260513-131457
HARBOR_REGISTRY=10.0.17.205
HARBOR_PROJECT=java-demo
HARBOR_REPOSITORY=java-demo
```

Result:

```text
Finished: SUCCESS
IMAGE_TAG=timeoutfix-20260513-131457
镜像地址=10.0.17.205/java-demo/java-demo:timeoutfix-20260513-131457
```

Harbor API confirmed the artifact exists:

```text
project=java-demo
repository=java-demo
tag=timeoutfix-20260513-131457
digest=sha256:dfd95f754c68440fa64114e61e27bea0fd1d2f7fb183875cf30c7b45e7d624ad
type=IMAGE
size=98564387
push_time=2026-05-13T05:27:44.808Z
```

### 5.4 Remaining E2E action

The failed AutoOps request is terminal and should not be reused:

```text
DR20260512163231.431627832 -> failed at build
```

Two duplicate requests are still pending approval and can be used for the next full AutoOps retry after confirming which one should proceed:

```text
DR20260512163248.423566394 -> pending_approval
DR20260512163337.185849092 -> pending_approval
```

After one of those is approved, the expected behavior is that Jenkins push no longer fails with Gateway 504 and AutoOps can continue to scan/deploy/notify.

## 6. Final E2E success — `DR20260513135942.298666965`

### 6.1 Pending approval cleanup before retry

Before the final retry, stale pending approval requests were removed so the DingTalk test could start cleanly. Historical succeeded / failed records were preserved.

```text
Deleted pending deploy_request rows:
- DR20260430145620.502662533
- DR20260512163248.423566394
- DR20260512163337.185849092

Verification: deploy_request pending / pending_approval rows = 0
Backup: /tmp/autoops-pending-approvals-20260513-135051.tsv
```

### 6.2 Real DingTalk request and approval

The real DingTalk flow produced a new request:

```text
requestNo=DR20260513135942.298666965
source=agent
workflowKind=build_deploy
mode=direct
namespace=ao-direct-java-demo-test
releaseName=java-demo
approvalStatus=approved
```

Initial retry failed in deploy stage because the target `Deployment/java-demo` already existed and the Direct deployer credential could create/delete but not update Deployments:

```text
stage=deploy
status=failed
error=应用 Deployment 失败: deployments.apps "java-demo" is forbidden: User "system:serviceaccount:autoops:autoops-deployer" cannot update resource "deployments" in API group "apps" in the namespace "ao-direct-java-demo-test"
```

### 6.3 Direct deployer RBAC tactical recovery

Root cause evidence:

```text
kubectl auth can-i update deployments.apps -n ao-direct-java-demo-test --as=system:serviceaccount:autoops:autoops-deployer -> no
kubectl auth can-i patch deployments.apps  -n ao-direct-java-demo-test --as=system:serviceaccount:autoops:autoops-deployer -> no
kubectl auth can-i create deployments.apps -n ao-direct-java-demo-test --as=system:serviceaccount:autoops:autoops-deployer -> yes
```

The live ClusterRole and GitOps chart were updated to keep the deployer credential durable. Commits in `/home/kchou/Code/pukka-gitops`:

```text
876c0f0 fix: allow AutoOps deployer to update direct resources
cecd2cc fix: allow AutoOps deployer to clean direct namespaces
```

ArgoCD confirmation:

```text
application=autoops
sync=Synced
health=Healthy
revisions=["cecd2cca2598ac275dc01aa400002c1e728450b9", "cecd2cca2598ac275dc01aa400002c1e728450b9"]
```

Final RBAC verification:

```text
update deployments.apps as autoops-deployer -> yes
update services as autoops-deployer         -> yes
delete namespaces as autoops-deployer       -> yes
```

### 6.4 Pipeline rerun and final AutoOps state

Because AutoOps has no dedicated retry/reset endpoint for a failed pipeline run, the approved failed request was reset to the existing scheduler contract after backing up the relevant tables:

```text
Backup: /tmp/autoops-retry-DR20260513135942-20260513-152342.sql
Reset: deploy_request requestStatus=approved executionStatus=pending pipelineStatus=pending
Reset: deploy_pipeline_run status=pending
Cleanup: old deploy_pipeline_stage_record rows deleted to avoid idx_pipeline_stage duplicate-key conflicts
```

Final AutoOps state:

```text
requestNo=DR20260513135942.298666965
requestStatus=succeeded
approvalStatus=approved
executionStatus=succeeded
pipelineStatus=succeeded
currentPipelineStage=deploy
runStatus=succeeded
currentStage=deploy
jenkinsBuildNumber=72
artifactTag=main-20260513-152618-c5bdf39
finalImageRef=10.0.17.205:80/java-demo/java-demo:main-20260513-152618-c5bdf39
lastError=
```

Stage records:

```text
build  succeeded external_id=72 started=2026-05-13T15:25:02+08:00 finished=2026-05-13T15:37:10+08:00
deploy succeeded               started=2026-05-13T15:37:10+08:00 finished=2026-05-13T15:37:24+08:00
notify succeeded               started=2026-05-13T15:37:25+08:00 finished=2026-05-13T15:37:25+08:00
```

### 6.5 Jenkins and Harbor evidence

Jenkins:

```text
job=java-demo-build
buildNumber=72
result=SUCCESS
artifactTag=main-20260513-152618-c5bdf39
```

Harbor / image evidence:

```text
finalImageRef=10.0.17.205:80/java-demo/java-demo:main-20260513-152618-c5bdf39
runningPodImageID=10.0.17.205:80/java-demo/java-demo@sha256:213d38af25c5f6b2d44163f6a7594322f34da71c4670d9230b0e50f928100e4b
```

The running pod image digest proves the image was pushed to Harbor and pulled by Kubernetes.

### 6.6 Kubernetes and NodePort access evidence

Kubernetes workload:

```text
namespace=ao-direct-java-demo-test
deployment=java-demo ready=1/1 image=10.0.17.205:80/java-demo/java-demo:main-20260513-152618-c5bdf39
service=java-demo type=NodePort port=80 targetPort=8080 nodePort=32580
pod=java-demo-6d4d9bf6fc-2xhp8 ready=1/1 status=Running node=pukka-k8s-wk03
```

NodePort checks:

```text
10.0.17.40:32580 -> 200 {"status":"ok","message":"AutoOps Java demo is running","application":"java-demo","profile":"test",...}
10.0.17.41:32580 -> 200 {"status":"ok","message":"AutoOps Java demo is running","application":"java-demo","profile":"test",...}
10.0.17.42:32580 -> 200 {"status":"ok","message":"AutoOps Java demo is running","application":"java-demo","profile":"test",...}
10.0.17.43:32580 -> 200 {"status":"ok","message":"AutoOps Java demo is running","application":"java-demo","profile":"test",...}
10.0.17.44:32580 -> 200 {"status":"ok","message":"AutoOps Java demo is running","application":"java-demo","profile":"test",...}
10.0.17.45:32580 -> 200 {"status":"ok","message":"AutoOps Java demo is running","application":"java-demo","profile":"test",...}
```

### 6.7 deploy_bot notification evidence

`deploy_notification` records for the final request:

```text
failed   sent dingtalk_robot 2026-05-13T14:23:33+08:00  # pre-fix failure notification
executed sent dingtalk_robot 2026-05-13T15:37:25+08:00  # final success notification
```

The final notification payload included the required user-facing fields:

```text
requestNo=DR20260513135942.298666965
status=succeeded
image=10.0.17.205:80/java-demo/java-demo:main-20260513-152618-c5bdf39
namespace=ao-direct-java-demo-test
serviceType=NodePort
servicePort=80 -> 8080
nodePort=32580
accessUrls=http://10.0.17.43:32580/, http://10.0.17.44:32580/, http://10.0.17.45:32580/, http://10.0.17.40:32580/, http://10.0.17.41:32580/, http://10.0.17.42:32580/
```

### 6.8 Final stop condition

The minimal E2E is complete:

```text
DingTalk user prompt -> Hermes -> AutoOps Agent API -> DingTalk OA approval -> Jenkins build #72 -> Harbor image -> AutoOps Direct Deployment/NodePort -> deploy_bot DingTalk notification
```

No secret values were recorded in this evidence file.
