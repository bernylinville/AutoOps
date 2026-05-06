# DingTalk AutoOps Pipeline — 会话交接文档 v3

**生成时间**: 2026-04-28 15:10 CST
**状态**: Build #20 正在运行 (Maven 依赖下载中)，Harbor Token 服务已修复，数据库种子已完成

---

## 一、本轮已完成

| # | 操作 | 结果 | 详情 |
|---|------|------|------|
| 1 | Jenkins GitLab 凭据更新 | ✅ | 新 PAT: `glpat-xxxxxxxxxxxxxxxxxxxx`，通过 Script Console 更新 |
| 2 | Build #19 失败根因分析 | ✅ | Maven BUILD SUCCESS，Jib Push Harbor 500 Error |
| 3 | Harbor Token 服务修复 | ✅ | PKCS#8→PKCS#1 RSA 格式私钥，patch `harbor-core-fixed` secret |
| 4 | Harbor Token 验证 | ✅ | `GET /service/token` 现在返回有效 JWT |
| 5 | 触发 Build #20 | ✅ | 参数: GIT_REF=main, RELEASE_NAME=java-demo-devtest, ENV=devtest |
| 6 | 数据库种子数据加载 | ✅ | sys_admin(5), k8s_cluster(1), deploy_cluster_target(2), app_application(1) |
| 7 | AutoOps captcha 验证 | ✅ | `http://10.0.17.206/api/v1/captcha` 返回 200 |
| 8 | Agent API Token 确认 | ✅ | `change-me-agent-token` 可用 (非 `dev-bearer-token`) |

---

## 二、Build #20 状态

**当前**: 仍在运行 (Maven 依赖下载阶段，K8s agent 无缓存)

**IMAGE_TAG 预期**: 会在 Jib 阶段输出类似 `20260428XXXXXX-41d7533` 的标签

**关键验证**: Build #20 的 Jib Push 可能仍然失败，因为 Jenkinsfile 使用 `harbor.harbor.svc.cluster.local` 作为 HARBOR_HOST，但 Jib 在 K8s agent pod 中运行时可能通过不同路径解析该地址。如果 Jib push 再次失败，需要检查:
1. K8s agent pod 是否能解析 `harbor.harbor.svc.cluster.local`
2. 如果不能，将 Jenkinsfile 的 HARBOR_HOST 改为 `10.233.29.94` (Harbor ClusterIP) 或 `10.0.17.205` (外部 IP)

### 检查命令
```bash
# Build #20 状态
curl -s -u "admin:pukka-jenkins" "http://10.0.17.204/job/java-demo-build/20/api/json?tree=building,result,duration" | python3 -m json.tool

# Build #20 控制台 (最后 80 行)
curl -s -u "admin:pukka-jenkins" "http://10.0.17.204/job/java-demo-build/20/consoleText" | tail -80

# 如果构建失败，检查 Jib 错误
curl -s -u "admin:pukka-jenkins" "http://10.0.17.204/job/java-demo-build/20/consoleText" | grep -i "error\|unauthorized\|FAILED"
```

---

## 三、Harbor Token 服务修复详情

### 根因
Harbor v2.12.2 的 token 服务无法解析 PKCS#8 格式的 `tls.key`:
```
[ERROR] [/core/service/token/token.go:50]: Unexpected error when creating the token, error: unable to get PrivateKey from PEM type: PRIVATE KEY
```

### 修复步骤
```bash
# 1. 提取 tls.key (PKCS#8 格式)
ssh -i ~/.ssh/id_pukka pukka@10.0.17.43 \
  "sudo kubectl --kubeconfig=/etc/kubernetes/admin.conf get secret harbor-core-fixed -n harbor -o jsonpath='{.data.tls\.key}'" | base64 -d > /tmp/harbor_tls_key_pkcs8.pem

# 2. 转换为 PKCS#1 RSA 格式
openssl rsa -in /tmp/harbor_tls_key_pkcs8.pem -traditional -out /tmp/harbor_tls_key_pkcs1.pem

# 3. Patch K8s secret
NEW_KEY_B64=$(cat /tmp/harbor_tls_key_pkcs1.pem | base64 -w0)
ssh -i ~/.ssh/id_pukka pukka@10.0.17.43 \
  "sudo kubectl --kubeconfig=/etc/kubernetes/admin.conf patch secret harbor-core-fixed -n harbor -p '{\"data\": {\"tls.key\": \"$NEW_KEY_B64\"}}'"

# 4. 重启 Harbor core
ssh -i ~/.ssh/id_pukka pukka@10.0.17.43 \
  "sudo kubectl --kubeconfig=/etc/kubernetes/admin.conf rollout restart deployment/harbor-core -n harbor"
```

### 验证
```bash
# Token 端点现在返回有效 JWT
curl -s -u 'robot$java-demo+jenkins-ci-v3:xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx' \
  'http://10.0.17.205/service/token?service=harbor-registry&scope=repository:java-demo/java-demo:pull,push'
# 期望: {"token":"eyJhbGci...","expires_in":1800,...}
```

---

## 四、数据库种子状态

| 表 | 记录数 | 详情 |
|----|--------|------|
| sys_admin | 5 | admin(ID=89), test(ID=102), zhangfan(105), lisi(106), wangwu(107) |
| k8s_cluster | 1 | pukka-k8s (ID=1, v1.30, type=1) |
| deploy_cluster_target | 2 | pukka-devtest(ID=2, env=devtest), pukka-staging(ID=3, env=staging) |
| app_application | 1 | java-demo (ID=3, code=java-demo) |

### ⚠️ 缺少的数据
- `k8s_cluster.credential` 为空 (需要 kubeconfig base64)
- `deploy_cluster_target.harbor_server_id` 和 `jenkins_server_id` 为空
- `deploy_cluster_target.default_approver_admin_id` 为空 (应设为 89)
- `config_account` 表未添加 Jenkins/Harbor 连接信息 (密码需要 AES 加密)

### 密码哈希
Admin 密码哈希为 bcrypt: `$2a$10$x/9pj5uUqQFg4Lse/4zpeu02EbHdczS5k01rQuFUSSfbBw7Ns8V4S`  
(这是 seed_data.sql 中的默认密码，原文未知，可能需要通过 AutoOps API 重置)

---

## 五、Agent API 部署请求格式

**端点**: `POST http://10.0.17.206/api/v1/integrations/agent/deploy-requests`
**认证**: `Authorization: Bearer change-me-agent-token`

### GitOps 模式请求体 (Build #20 成功后使用):
```json
{
  "requesterExternalType": "dingtalk",
  "requesterExternalId": "hermes-test-user",
  "requesterDisplayName": "Hermes测试用户",
  "mode": "gitops",
  "workflowKind": "deploy_only",
  "resourceType": "deployment",
  "clusterTargetId": 2,
  "releaseName": "java-demo-devtest",
  "namespace": "ao-gitops-java-demo-devtest",
  "image": "harbor.harbor.svc.cluster.local/java-demo/java-demo:<IMAGE_TAG>",
  "replicas": 1,
  "reason": "钉钉触发部署验证",
  "approverAdminId": 89
}
```

### Direct 模式请求体:
```json
{
  "requesterExternalType": "dingtalk",
  "requesterExternalId": "hermes-test-user",
  "requesterDisplayName": "Hermes测试用户",
  "mode": "direct",
  "resourceType": "deployment",
  "clusterTargetId": 2,
  "releaseName": "java-demo-devtest",
  "namespace": "ao-gitops-java-demo-devtest",
  "image": "harbor.harbor.svc.cluster.local/java-demo/java-demo:<IMAGE_TAG>",
  "replicas": 1,
  "reason": "钉钉触发部署验证"
}
```

> 注意: `clusterTargetId` = 2 (pukka-devtest)，`approverAdminId` = 89 (admin)
> IMAGE_TAG 需要等 Build #20 完成后从 Jenkins 控制台获取

---

## 六、下一步操作清单

1. **等待 Build #20 完成** — 检查 Jib push 阶段是否成功
   ```bash
   curl -s -u "admin:pukka-jenkins" "http://10.0.17.204/job/java-demo-build/20/api/json?tree=building,result,duration"
   ```

2. **如果 Jib push 仍然失败 (K8s agent 网络问题)** — 修改 Jenkinsfile HARBOR_HOST
   - 将 `harbor.harbor.svc.cluster.local` 改为 Harbor ClusterIP `10.233.29.94`
   - 或改为外部 IP `10.0.17.205` 并确保 `allowInsecureRegistries=true` 已设置

3. **验证 Harbor 镜像 + Trivy 扫描**
   ```bash
   # 检查镜像仓库
   curl -s -u "admin:pukka-harbor" "http://10.0.17.205/api/v2.0/projects/java-demo/repositories" | python3 -m json.tool
   
   # 检查特定镜像标签
   curl -s -u "admin:pukka-harbor" "http://10.0.17.205/api/v2.0/projects/java-demo/repositories/java-demo/artifacts" | python3 -m json.tool
   ```

4. **补充数据库: kubeconfig, Jenkins/Harbor config, approver**
   ```sql
   -- 更新 k8s_cluster 添加 kubeconfig
   -- 获取 kubeconfig: ssh -i ~/.ssh/id_pukka pukka@10.0.17.43 "sudo cat /etc/kubernetes/admin.conf" | base64 -w0
   UPDATE k8s_cluster SET credential='<base64_kubeconfig>' WHERE id=1;
   
   -- 更新 deploy_cluster_target approver
   UPDATE deploy_cluster_target SET default_approver_admin_id=89 WHERE id IN (2,3);
   ```

5. **通过 Agent API 触发部署** (在 Build #20 成功 + Harbor 镜像验证后)

6. **验证 K8s 部署**
   ```bash
   ssh -i ~/.ssh/id_pukka pukka@10.0.17.43 "sudo kubectl --kubeconfig=/etc/kubernetes/admin.conf get pods -n ao-gitops-java-demo-devtest"
   ```

---

## 七、凭证汇总

| 服务 | 地址 | 用户名 | 密码/Token |
|------|------|--------|------------|
| Jenkins | http://10.0.17.204 | admin | `pukka-jenkins` |
| Harbor | http://10.0.17.205 | admin | `pukka-harbor` |
| Harbor robot | — | `robot$java-demo+jenkins-ci-v3` | `xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx` |
| GitLab PAT | http://gayhub.seeingtv.com | kchou | `glpat-xxxxxxxxxxxxxxxxxxxx` |
| Volces ACR | — | `crrobot@pukka_images` | `<your-password>` |
| AutoOps Agent | http://10.0.17.206 | Bearer | `change-me-agent-token` |
| PostgreSQL | localhost:15432 | devops | `devops@2025` |
| K8s SSH | `pukka@10.0.17.43` | — | `~/.ssh/id_pukka`, sudo 免密 |

---

## 八、关键文件路径

- Jenkinsfile: `/tmp/java-demo/Jenkinsfile`
- GitOps 仓库: `~/Code/pukka-gitops/`
- AutoOps 仓库: `~/Code/AutoOps/`
- 集群访问: `~/Code/pukka-gitops/CLUSTER-ACCESS.md`
- Harbor PKCS#1 key: `/tmp/harbor_tls_key_pkcs1.pem`
- Seed SQL: `~/Code/AutoOps/docker/postgres/seed_data.sql`
- 镜像同步脚本: `~/Code/pukka-gitops/scripts/mirror-images.sh`

---

## 九、AutoOps API 参考

```
# AutoOps 登录 (captcha 先获取)
GET  /api/v1/captcha                                          # → 200 OK
POST /api/v1/login  {"username":"admin","password":"...","captchaId":"...","captchaCode":"..."}

# Agent 部署请求
POST /api/v1/integrations/agent/deploy-requests               # → 创建部署请求
GET  /api/v1/integrations/agent/deploy-requests/:requestNo    # → 查询状态
POST /api/v1/integrations/agent/deploy-requests/:requestNo/execute  # → 执行部署

# K8s 集群管理 (UI)
POST /api/v1/k8s/cluster                                      # → 创建集群
```

---

## 十、已知问题 & 注意事项

1. **Jenkins 使用 K8s agent pod** (maven-jdk21) — 需要验证 pod 是否能解析 `harbor.harbor.svc.cluster.local`
2. **Harbor Token 500** 是 PKCS#8 vs PKCS#1 问题 — 已修复但需确认 K8s 内部 DNS 解析
3. **数据库 sys_admin 密码** — seed_data.sql 中的 bcrypt 哈希对应什么密码未知，可能需要通过 API 重置
4. **Jenkinsfile HARBOR_HOST** — 当前设为 `harbor.harbor.svc.cluster.local`，如果 K8s agent 无法解析，需改为 IP
5. **GitOps 图像标签** — Jenkinsfile 用 `${ts}-${gitShort}` 生成标签，不会 export 为环境变量，`post { success }` 块中的 IMAGE_TAG 变量可能为空