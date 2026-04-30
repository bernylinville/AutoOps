# DingTalk AutoOps Pipeline — 会话交接文档 v2

**生成时间**: 2026-04-28 15:00 CST
**状态**: Harbor Token 服务修复中，Jenkins Build #19 Maven 成功但 Jib Push 失败

---

## 一、本轮已完成

| # | 操作 | 结果 |
|---|------|------|
| 1 | Jenkins GitLab 凭据更新为 `glpat-xxxxxxxxxxxxxxxxxxxx` | ✅ 通过 Groovy Script Console 更新成功 |
| 2 | 触发 Build #19 | ✅ Checkout + Maven Build SUCCESS (4m27s) |
| 3 | 定位 Harbor 500 根因 | ✅ Harbor token 服务 PKCS#8 私钥不兼容 |
| 4 | 修复 Harbor TLS 私钥 (PKCS#8→PKCS#1) | ✅ 已 patch secret + restart core pod |
| 5 | AutoOps captcha 可用 | ✅ `http://10.0.17.206/api/v1/captcha` 返回 200 |
| 6 | AutoOps Agent API token 确认 | ✅ `change-me-agent-token` 可用 (非 `dev-bearer-token`) |
| 7 | 数据库种子状态检查 | ⚠️ k8s_cluster / deploy_cluster_target / app_application / sys_admin 全部为空 |

---

## 二、Build #19 详细状态

### Maven 阶段: ✅ SUCCESS
```
BUILD SUCCESS
Total time: 04:27 min
Finished at: 2026-04-28T06:55:47Z
IMAGE_TAG=20260428065603-41d7533
```

### Jib Push 阶段: ❌ FAILURE
```
Unauthorized for harbor.harbor.svc.cluster.local/java-demo/java-demo: 500 Internal Server Error
GET http://10.0.17.205/service/token?service=harbor-registry&scope=repository:java-demo/java-demo:pull,push
```

**根因**: Harbor token 服务 (`/service/token`) 无法用 PKCS#8 格式的 `tls.key` 签发 JWT token。这在 Harbor v2.12.2 是已知问题。

### 修复动作
```bash
# 从 K8s secret 提取 tls.key (PKCS#8 格式)
# 转换为 PKCS#1 RSA 格式
openssl rsa -in /tmp/harbor_tls_key_pkcs8.pem -traditional -out /tmp/harbor_tls_key_pkcs1.pem

# Patch K8s secret
kubectl patch secret harbor-core-fixed -n harbor -p '{"data": {"tls.key": "<base64-pkcs1-key>"}}'

# Restart Harbor core
kubectl rollout restart deployment/harbor-core -n harbor
# 新 pod harbor-core-74544b8b8-pgf2p 已 Running
```

### ⚠️ 待验证: Harbor token 服务是否修复成功
**需要运行**:
```bash
# 验证 token 端点
curl -s "http://10.0.17.205/service/token?service=harbor-registry&scope=repository:java-demo/java-demo:pull,push" \
  -u 'robot$java-demo+jenkins-ci-v3:xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx'
# 期望: 返回 JSON token (不再是 500)

# 如果修复成功，重新触发 Build #20
CRUMB=$(curl -s -c /tmp/cookies -u "admin:pukka-jenkins" "http://10.0.17.204/crumbIssuer/api/json" | python3 -c "import sys,json; print(json.load(sys.stdin)['crumb'])")
curl -s -b /tmp/cookies -u "admin:pukka-jenkins" -X POST \
  "http://10.0.17.204/job/java-demo-build/buildWithParameters?GIT_REF=main&RELEASE_NAME=java-demo-devtest&ENV=devtest" \
  -H "Jenkins-Crumb: $CRUMB"

# 等待构建完成后检查
curl -s -u "admin:pukka-jenkins" "http://10.0.17.204/job/java-demo-build/lastBuild/api/json?tree=building,result,duration" | python3 -m json.tool
```

---

## 三、数据库种子数据 — 待填充

数据库核心表全部为空，需要种子数据才能使用 Agent API 和 UI。

### 必须创建的记录

```sql
-- 1. 系统管理员 (用于 UI 登录)
INSERT INTO sys_admin (username, password, nickname, status)
VALUES ('admin', '$2a$10$...bcrypt_hash...', '管理员', 1);
-- 注意: 密码需要 bcrypt 哈希，根据 AutoOps 代码的密码哈希方式生成

-- 2. K8s 集群
INSERT INTO k8s_cluster (name, version, cluster_type, credential, node_count, ready_nodes, master_nodes, worker_nodes)
VALUES ('pukka-k8s', '1.30', 1, '<kubeconfig_base64>', 6, 6, 3, 3);
-- 已插入 ID=1, 需要补充 credential (kubeconfig)

-- 3. 部署集群目标 (devtest + staging)
INSERT INTO deploy_cluster_target (name, kube_cluster_id, env_type, git_ops_enabled, direct_enabled, harbor_server_id, jenkins_server_id, git_ops_repo, git_ops_branch, git_ops_release_dir, direct_namespace_prefix, default_ttl_hours, default_approver_admin_id)
VALUES
  ('pukka-devtest', 1, 'devtest', true, true, NULL, NULL, 'http://gayhub.seeingtv.com/ipaas/pukka-gitops.git', 'main', 'apps', 'ao-direct', 72, NULL),
  ('pukka-staging', 1, 'staging', true, true, NULL, NULL, 'http://gayhub.seeingtv.com/ipaas/pukka-gitops.git', 'main', 'apps', 'ao-direct', 168, NULL);

-- 4. 应用注册
INSERT INTO app_application (name, code, business_group_id, business_dept_id, repo_url, programming_lang)
VALUES ('java-demo', 'java-demo', 1, 1, 'http://gayhub.seeingtv.com/ipaas/java-demo.git', 'Java');
```

⚠️ 实际插入前需要确认:
- sys_admin 的 bcrypt 密码哈希方式 (查看 AutoOps 代码 `api/api/system/service/`)
- business_group 和 business_dept 是否需要先创建
- Harbor server 和 Jenkins server 是否需要注册
- kubeconfig 作为 credential 的格式

---

## 四、Agent API 部署请求格式

**端点**: `POST /api/v1/integrations/agent/deploy-requests`
**认证**: `Authorization: Bearer change-me-agent-token`

### 请求体 (gitops 模式):
```json
{
  "requesterExternalType": "dingtalk",
  "requesterExternalId": "dingtalk_user_id",
  "requesterDisplayName": "测试用户",
  "mode": "gitops",
  "workflowKind": "build_deploy",
  "resourceType": "deployment",
  "clusterTargetId": <devtest_target_id>,
  "releaseName": "java-demo-devtest",
  "namespace": "ao-gitops-java-demo-devtest",
  "image": "harbor.harbor.svc.cluster.local/java-demo/java-demo:20260428065603-41d7533",
  "replicas": 1,
  "reason": "DingTalk 触发部署",
  "gitRef": "main",
  "harborProject": "java-demo",
  "harborRepository": "java-demo",
  "artifactTag": "20260428065603-41d7533",
  "approverAdminId": 89
}
```

---

## 五、集群凭证

| 服务 | 地址 | 用户名 | 密码/Token |
|------|------|--------|------------|
| Jenkins | http://10.0.17.204 | admin | `pukka-jenkins` |
| Harbor | http://10.0.17.205 | admin | `pukka-harbor` |
| Harbor robot | — | `robot$java-demo+jenkins-ci-v3` | `xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx` |
| GitLab PAT | http://gayhub.seeingtv.com | kchou | `glpat-xxxxxxxxxxxxxxxxxxxx` |
| Volces ACR | — | `crrobot@pukka_images` | `<your-password>` |
| AutoOps Agent API | http://10.0.17.206 | Bearer | `change-me-agent-token` |
| K8s API | SSH `pukka@10.0.17.43` | — | `sudo kubectl --kubeconfig=/etc/kubernetes/admin.conf` |
| PostgreSQL | localhost:15432 | devops | `devops@2025` |

---

## 六、下一步操作清单

1. **[阻塞]** 验证 Harbor token 服务修复 — `curl http://10.0.17.205/service/token?...`
2. **[阻塞]** 触发 Build #20 — Harbor 修复后重新构建
3. **[阻塞]** 验证 Harbor 镜像推送 + Trivy 扫描
4. 种子数据库 (sys_admin, k8s_cluster credential, deploy_cluster_target, app_application)
5. AutoOps UI 登录验证 (captcha 已 OK)
6. 通过 Agent API 触发部署请求
7. 验证 K8s 部署: `kubectl get pods -n ao-gitops-java-demo-devtest`
8. Dowktalk Hermes E2E 测试

---

## 七、Harbor 修复关键文件

Harbor core 部署在 K8s namespace `harbor`，token 签名私钥来自:
- Secret: `harbor-core-fixed`
- Key: `tls.key`
- 挂载路径: `/etc/core/private_key.pem` (subPath: `tls.key`)
- **修复**: 将 PKCS#8 `-----BEGIN PRIVATE KEY-----` 转为 PKCS#1 `-----BEGIN RSA PRIVATE KEY-----`

```bash
# 本地已有转换后的文件
ls -la /tmp/harbor_tls_key_pkcs1.pem  # PKCS#1 RSA 格式
ls -la /tmp/harbor_tls_key_pkcs8.pem  # 原始 PKCS#8 格式

# SSH 访问集群
ssh -i ~/.ssh/id_pukka pukka@10.0.17.43

# Harbor core pod 状态
kubectl --kubeconfig=/etc/kubernetes/admin.conf get pods -n harbor -l component=core
```

---

## 八、Jenkins Build #19 Console 完整日志关键部分

```
[INFO] BUILD SUCCESS
[INFO] Total time:  04:27 min
IMAGE_TAG=20260428065603-41d7533

+ ./mvnw -B com.google.cloud.tools:jib-maven-plugin:3.4.1:build \
  -Djib.from.image=pukka-all-images-cn-shanghai.cr.volces.com/proxy/eclipse-temurin:21-jre \
  -Djib.to.image=harbor.harbor.svc.cluster.local/java-demo/java-demo:20260428065603-41d7533
[ERROR] Unauthorized for harbor.harbor.svc.cluster.local/java-demo/java-demo: 500 Internal Server Error
[ERROR] GET http://10.0.17.205/service/token?service=harbor-registry&scope=repository:java-demo/java-demo:pull,push
```