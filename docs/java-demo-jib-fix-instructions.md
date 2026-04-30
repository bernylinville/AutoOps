# java-demo Jib HTTP 凭据问题修复指南

## 1. 问题根因

### 1.1 Jib 安全策略

Jib（Google 开源的容器镜像构建工具）默认配置下，**拒绝通过明文 HTTP 传输 Registry 凭据**。当目标 Registry 地址采用 `http://` 协议时，Jib 会抛出错误：

```
Error occurred while pushing image to registry: credentials over plain HTTP
```

### 1.2 为何不是 Jenkins RBAC 问题

- Jenkins 权限配置（RBAC）控制的是用户对 Jenkins 作业的操作权限，与镜像推送无关
- Harbor 权限（project member 角色）控制的是用户对 Harbor 项目的访问权限
- **这个错误来自 Jib 内部的安全检查**，在 Maven build 阶段就已拒绝，不涉及 Jenkins / Harbor 身份验证

---

## 2. 修复方案 A：Jib 参数（推荐，快速生效）

### 2.1 Jenkinsfile 中修改 Maven 命令

**修改前：**
```groovy
stage('Build & Push') {
    steps {
        sh '''
            cd java-demo
            mvn clean package -Djib.to.image=10.0.17.205/library/java-demo:${BUILD_NUMBER}
        '''
    }
}
```

**修改后：**
```groovy
stage('Build & Push') {
    steps {
        sh '''
            cd java-demo
            mvn clean package \
              -Djib.to.image=10.0.17.205/library/java-demo:${BUILD_NUMBER} \
              -Djib.to.auth.sendCredentialsOverHttp=true
        '''
    }
}
```

### 2.2 或者在 pom.xml 中配置 Jib 插件

在 `pom.xml` 的 `<build><plugins>` 中找到 `jib-maven-plugin`，添加或修改配置：

```xml
<plugin>
    <groupId>com.google.cloud.tools</groupId>
    <artifactId>jib-maven-plugin</artifactId>
    <version>3.4.0</version>
    <configuration>
        <from>
            <image>eclipse-temurin:21-jdk-alpine</image>
        </from>
        <to>
            <image>10.0.17.205/library/java-demo</image>
            <auth>
                <username>${env.HARBOR_USER}</username>
                <password>${env.HARBOR_PASS}</password>
            </auth>
        </to>
        <!-- 允许不安全的 HTTP Registry -->
        <allowInsecureRegistries>true</allowInsecureRegistries>
    </configuration>
</plugin>
```

### 2.3 确保 Jenkins Credentials 配置

在 Jenkins UI 中，确认 Harbor 凭据已注入为环境变量：

```groovy
withCredentials([usernamePassword(
    credentialsId: 'harbor-credentials',
    usernameVariable: 'HARBOR_USER',
    passwordVariable: 'HARBOR_PASS'
)]) {
    sh '''
        cd java-demo
        mvn clean package \
          -Djib.to.image=10.0.17.205/library/java-demo:${BUILD_NUMBER} \
          -Djib.to.auth.sendCredentialsOverHttp=true
    '''
}
```

### 2.4 技术债说明

这是**临时方案**，存在以下风险：

- Harbor 凭据通过明文 HTTP 传输，容易被网络嗅探
- 违反 zero-trust 安全原则
- 不符合生产环境最佳实践

---

## 3. 修复方案 B：Harbor 切换 HTTPS（长期方案）

### 3.1 方案概述

使用 Kubernetes Ingress + cert-manager，为 Harbor 配置 TLS 证书：

- Ingress hostname：`harbor.example.com` (或内部域名)
- TLS certificate：由 cert-manager 自动颁发 (Let's Encrypt 或自签)
- Jib 将自动接受 HTTPS 地址，无需 `allowInsecureRegistries` 开关

### 3.2 计划与负责人

| 项目 | 时间 | 负责人 | 状态 |
|------|------|--------|------|
| E2E 功能打通 | 当前 | - | 进行中 |
| Harbor HTTPS 迁移 | E2E 后 2 周内 | 待定 | 未开始 |

### 3.3 准备工作（暂不执行）

- Harbor 部署在 K8s (harbor.harbor.svc.cluster.local)，已具备 Ingress 基础
- 集群已安装 cert-manager，可自动签发证书
- 详细步骤参考 Harbor 官方文档：[Harbor Ingress Setup](https://goharbor.io/docs/)

---

## 4. 验证步骤

### 4.1 重新触发 Jenkins Build

```bash
# 在 Jenkins UI 中手动点击 "Build Now"，或通过 CLI：
curl -X POST http://10.0.17.204/job/java-demo/build \
  --user jenkins-user:jenkins-token
```

### 4.2 检查 Console Output

在 Jenkins Build #21 (或后续) 的 Console Output 中，确认：

- ✅ 不再出现 "credentials over plain HTTP" 错误
- ✅ 镜像成功推送到 Harbor：
  ```
  Built and pushed image as 10.0.17.205/library/java-demo:21
  Digest: sha256:abc123...
  ```

### 4.3 验证 Harbor 仓库

1. 打开 Harbor UI：http://10.0.17.205
2. 导航到 **Projects** > **library** > **java-demo**
3. 确认新 artifact 出现（tag 为 `21` 或对应 BUILD_NUMBER）

### 4.4 确认 Vulnerability Scan

Harbor 已预装默认 scanner，新 artifact 应自动扫描：

1. 在 Harbor UI 中点击该 artifact
2. 查看 **Vulnerabilities** 标签页
3. 确认扫描结果已生成（绿色或黄色，取决于漏洞数量）

---

## 5. 技术债记录

### 5.1 登记信息

| 字段 | 值 |
|------|-----|
| 问题 | Jib HTTP 明文传输凭据（`allowInsecureRegistries=true`） |
| 影响 | java-demo 镜像构建流程 |
| 根因 | Harbor 未配置 HTTPS |
| 临时方案 | 添加 `-Djib.to.auth.sendCredentialsOverHttp=true` 参数 |
| 截止日期 | E2E 功能打通后 2 周内 |
| 长期方案 | Harbor Ingress + TLS 证书 |
| 负责人 | **待定（修复时填写）** |
| 优先级 | P1（影响构建流程） |

### 5.2 完成清单

- [ ] 修复方案 A 已在 Jenkinsfile 中应用
- [ ] Jenkins Build #21+ 成功执行，Harbor 仓库中出现新 artifact
- [ ] Harbor vulnerability scan 自动触发
- [ ] 技术债负责人已确认
- [ ] Harbor HTTPS 迁移计划已纳入 v1.1 或 v2.0 roadmap
- [ ] 原 `allowInsecureRegistries` 参数已在 deadline 前移除

---

## 参考资源

- [Google Jib 官方文档](https://github.com/GoogleContainerTools/jib)
- [Jib Maven Plugin Configuration](https://github.com/GoogleContainerTools/jib/tree/master/jib-maven-plugin)
- [Harbor 官方文档](https://goharbor.io/docs/)
- [Harbor 安全最佳实践](https://goharbor.io/docs/2.10/administration/configure-https/)
