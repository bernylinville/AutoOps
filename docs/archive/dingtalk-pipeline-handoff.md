# DingTalk AutoOps Pipeline — 执行上下文交接

**生成时间**: 2026-04-28
**状态**: Jenkins 构建链接近完成，Harbor 推送待验证

---

## 一、场景目标

钉钉群@机器人发送自然语言构建需求 → Hermes Agent → AutoOps API → Jenkins Pipeline → Harbor 扫描 → GitOps 部署 → 结果返回

## 二、开发/测试环境

- 两套环境（devtest / staging），通过 K8s namespace 区分
- AutoOps 编排调度，Jenkinsfile 控制 CI/CD
- GitLab: `http://gayhub.seeingtv.com/ipaas/java-demo.git`

## 三、Chrome DevTools 验证结论

| 组件 | 地址 | 发现 |
|------|------|------|
| **Jenkins** | http://10.0.17.204 | admin 完全权限 ("Logged-in users can do anything")，403 是 CSRF 不是权限 |
| **Harbor** | http://10.0.17.205 | Trivy v0.58.2 Healthy，java-demo 项目存在，robot 已创建 |
| **AutoOps** | http://10.0.17.206 | captcha 服务不可用，无法登录 UI |

## 四、jenkins-0 pod 网络验证

```
# 已验证 jenkins-0 pod 可达 Harbor (所有地址 200):
curl http://10.233.29.94/api/v2.0/health     # ClusterIP ✅
curl http://10.0.17.205/api/v2.0/health       # External ✅
curl http://10.233.115.99:8080/api/v2.0/health # Direct ✅
```

**但 K8s agent pod 无法访问 Harbor（Calico CNI / Jenkins 插件网络隔离）**

## 五、已完成的修复

| 问题 | 解决方案 |
|------|----------|
| gitlab-credentials PAT 过期 | 更新为 `glpat-xxxxxxxxxxxxxxxxxxxx` |
| git dubious ownership | Jenkinsfile 添加 `safe.directory "*"` |
| Harbor robot 凭据无效 | Chrome DevTools 创建新 robot `jenkins-ci-v3` (70 权限) |
| Eclipse Temurin 基础镜像不可用 | `docker push` 到火山云 ACR `proxy/eclipse-temurin:21-jre` |
| Jenkinsfile Groovy/env 变量冲突 | 使用 `\${}` shell escape |

## 六、当前 Jenkinsfile

```groovy
pipeline {
    agent any   // 使用 built-in 节点 (jenkins-0, 已验证可达 Harbor)
    environment {
        HARBOR_HOST = "harbor.harbor.svc.cluster.local"
        HARBOR_PROJECT = "java-demo"
        HARBOR_USER = "robot\$java-demo+jenkins-ci-v3"
        HARBOR_PASS = "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
    }
    parameters {
        string(name: 'GIT_REF', defaultValue: 'main')
        string(name: 'RELEASE_NAME', defaultValue: 'java-demo-devtest')
        string(name: 'ENV', defaultValue: 'devtest')
    }
    stages {
        stage('Checkout') { steps { checkout scm } }
        stage('Build & Test') { steps { sh './mvnw -B clean verify' } }
        stage('Build & Push Image') {
            steps {
                script {
                    sh 'git config --global --add safe.directory "*"'
                    def gitShort = sh(script: 'git rev-parse --short=7 HEAD', returnStdout: true).trim()
                    def ts = sh(script: "date -u +'%Y%m%d%H%M%S'", returnStdout: true).trim()
                    sh """
                        ./mvnw -B com.google.cloud.tools:jib-maven-plugin:3.4.1:build \
                          -Djib.from.image="pukka-all-images-cn-shanghai.cr.volces.com/proxy/eclipse-temurin:21-jre" \
                          -Djib.from.auth.username="crrobot@pukka_images" \
                          -Djib.from.auth.password="<your-password>" \
                          -Djib.to.image="\${HARBOR_HOST}/\${HARBOR_PROJECT}/java-demo:${ts}-${gitShort}" \
                          -Djib.to.auth.username="\${HARBOR_USER}" \
                          -Djib.to.auth.password="\${HARBOR_PASS}" \
                          -Djib.allowInsecureRegistries=true \
                          -DskipTests -q
                    """
                }
            }
        }
    }
    post { success { echo "IMAGE_TAG=${IMAGE_TAG}" } }
}
```

## 七、待验证/完成

1. **Build #18 完成** — agent any + Volces ACR proxy + Harbor ClusterIP
2. Harbor 镜像扫描验证
3. GitOps 部署到 devtest namespace
4. staging 环境部署
5. AutoOps captcha 修复（UI 登录审批人管理）
6. Hermes/DingTalk 自然语言 E2E 测试
7. 审批人白名单已配置 (approver_admin_id=89 for java-demo)

## 八、集群凭证

| 服务 | 用户名 | 密码/Token |
|------|--------|------------|
| Jenkins | admin | `pukka-jenkins` |
| Harbor | admin | `pukka-harbor` |
| Harbor robot | `robot$java-demo+jenkins-ci-v3` | `xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx` |
| GitLab PAT | — | `glpat-xxxxxxxxxxxxxxxxxxxx` |
| Volces ACR | `crrobot@pukka_images` | `<your-acr-password>` |
| k8s kubeconfig | SSH `pukka@10.0.17.43` | `sudo cat /etc/kubernetes/admin.conf` |

## 九、过度设计评估

v7 计划 (891 行) 中大部分 W4.x 后端变更已实现并部署。核心阻塞是 **Jenkins K8s agent pod 网络隔离** 和 **Harbor 500 内部错误**，不是代码缺失。

## 十、关键文件路径

- Jenkinsfile: `/tmp/java-demo/Jenkinsfile`
- gitops repo: `~/Code/pukka-gitops/`
- AutoOps repo: `~/Code/AutoOps/`
- opsclaw Hermes: `ssh opsclaw` → `~/.hermes/`
