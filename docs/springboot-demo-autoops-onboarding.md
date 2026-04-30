# Spring Boot Demo → GitLab → Jenkins → Harbor → AutoOps Direct 部署指引

更新时间：2026-04-29

目标：准备一个不依赖数据库/Redis 的 Spring Boot demo，push 到 GitLab 后，只需在钉钉/Hermes 对话里提供 GitLab 地址，AutoOps 即可自动接入应用与 dev/test Profile，触发共享 Jenkins 参数化 Job 构建镜像、推送 Harbor，并通过 AutoOps Direct 部署到 K8s。

## 1. 推荐 demo 形态

使用 Spring Initializr 新建最小项目：

- Project：Maven
- Language：Java
- Spring Boot：当前稳定 3.x
- Java：21（如 Jenkins 只有 JDK 17，则选 17）
- Dependencies：Spring Web、Spring Boot Actuator

最小代码示例：

```java
package com.example.demo;

import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RestController;

@SpringBootApplication
public class DemoApplication {
    public static void main(String[] args) {
        SpringApplication.run(DemoApplication.class, args);
    }
}

@RestController
class DemoController {
    @GetMapping("/")
    public String hello() {
        return "hello from autoops springboot demo";
    }
}
```

`src/main/resources/application.yml`：

```yaml
server:
  port: 8080
management:
  endpoints:
    web:
      exposure:
        include: health,info
```

## 2. 推送到 GitLab

示例：

```bash
git init
git add .
git commit -m "Bootstrap Spring Boot AutoOps demo"
git branch -M main
git remote add origin git@gayhub.seeingtv.com:demo/springboot-demo.git
git push -u origin main
```

仓库地址支持两种格式：

- `git@gayhub.seeingtv.com:demo/springboot-demo.git`
- `https://gayhub.seeingtv.com/demo/springboot-demo.git`

> AutoOps 会校验 Git host 是否在 `integrations.agent.project_onboarding.allowed_git_hosts` 中，避免机器人接入未知来源代码。

## 3. AutoOps 管理员一次性配置

在 `api/config.yaml` 中启用项目自动接入：

```yaml
integrations:
  agent:
    bearer_token: "<agent-token>"
    project_onboarding:
      enabled: true
      allowed_git_hosts:
        - gayhub.seeingtv.com
      shared_jenkins_job_name: autoops-springboot-build
      default_business_group_id: 1
      default_business_dept_id: 1
      default_jenkins_server_id: 3
      default_harbor_server_id: 4
      default_harbor_project: library
      default_approver_admin_id: 9
      dev_cluster_target_id: 7
      test_cluster_target_id: 8
      namespace_prefix: ao-direct
      default_service_port: 80
      default_target_port: 8080
      nodeport_access_host: "<k8s-node-ip-or-vip>"
```

这些 ID 均来自 AutoOps 已有数据：业务组、部门、Jenkins/Harbor 凭据、审批人、dev/test ClusterTarget。配置缺失时，新接口会返回明确错误，不会向普通用户索要镜像、namespace、Jenkins 或 Harbor 信息。

共享 Jenkins Job 至少需要接收以下参数：

| 参数 | 含义 |
|---|---|
| `GIT_URL` / `GIT_REPO_URL` | GitLab 仓库地址 |
| `GIT_REF` | 分支/tag/commit，用户未指定时为 `main` |
| `APPLICATION_CODE` | AutoOps 应用代号 |
| `ENV` | `dev` 或 `test` |
| `HARBOR_PROJECT` | Harbor 项目 |
| `HARBOR_REPOSITORY` | Harbor 仓库 |
| `RELEASE_NAME` | K8s release/service 名称 |
| `SERVICE_PORT` / `TARGET_PORT` | Service 与容器端口 |

Jenkins Console 需要输出可被 AutoOps 识别的镜像标签，例如：

```text
IMAGE_TAG=main-20260429-abcdef
```

## 4. Hermes 对话方式

### NodePort 对外访问

```text
@机器人 接入并部署 git@gayhub.seeingtv.com:demo/springboot-demo.git 到开发环境，分支 main，暴露 nodeport
```

Hermes 调用：

```http
POST /api/v1/integrations/agent/project-onboard-build-deploy
```

请求体：

```json
{
  "requesterExternalType": "dingtalk",
  "requesterExternalId": "<sender-userid>",
  "requesterDisplayName": "<sender-name>",
  "gitRepoUrl": "git@gayhub.seeingtv.com:demo/springboot-demo.git",
  "env": "dev",
  "gitRef": "main",
  "exposureMode": "nodeport",
  "reason": "接入并部署 springboot-demo 到开发环境，暴露 nodeport"
}
```

### ClusterIP 集群内访问

```text
@机器人 接入并部署 https://gayhub.seeingtv.com/demo/springboot-demo.git 到 test，clusterip
```

`exposureMode=clusterip` 或省略时，AutoOps 创建 `ClusterIP` Service。

## 5. 当前暴露模式范围

| exposureMode | 当前状态 | 说明 |
|---|---|---|
| `clusterip` | 支持 | 集群内访问，默认值 |
| `nodeport` | 支持 | 创建 NodePort Service；实际 nodePort 由 K8s 分配 |
| `gateway` | 暂不支持 | API 明确拒绝，后续版本扩展 |
| `metallb` | 暂不支持 | API 明确拒绝，后续版本扩展 |

v1.1 状态接口的 `accessInfo` 返回 `serviceType/servicePort/targetPort`，暂不返回实时分配的 `nodePort`。真实访问地址可先通过：

```bash
kubectl -n ao-direct-springboot-demo get svc springboot-demo
curl http://<nodeport_access_host>:<nodePort>/
```

## 6. 成功链路

1. Hermes 解析 GitLab 地址、env、gitRef、exposureMode。
2. AutoOps 校验 Git host allowlist。
3. AutoOps 自动创建或复用 `Application(code=springboot-demo)`。
4. AutoOps 自动创建或复用 dev/test `AppDeployProfile`，namespace 默认为 `ao-direct-springboot-demo`。
5. AutoOps 发起 build_deploy 申请与钉钉 OA 审批。
6. 审批通过后 Jenkins 构建镜像并推 Harbor。
7. Harbor 扫描通过后 AutoOps Direct 调 K8s API 创建 Deployment + Service。
8. NodePort 模式下，通过 `<nodeport_access_host>:<nodePort>` 访问 demo。

## 7. 常见失败与处理

| 错误 | 原因 | 处理 |
|---|---|---|
| `project_onboarding.enabled 未开启` | AutoOps 未启用自动接入 | 管理员补配置并重启 API |
| `GitLab host 不在允许列表` | 仓库不属于允许 GitLab | 检查 `allowed_git_hosts` |
| `应用代号已存在，但仓库地址不同` | code 冲突，避免误部署 | 指定新的 `applicationCode` 或管理员确认现有应用 |
| `部署目标未配置 directKubeconfigRef` | ClusterTarget 无 Direct 凭据 | 在 ClusterTarget 配置 kubeconfig 引用 |
| `当前自动接入 MVP 仅支持 clusterip/nodeport` | 用户要求 gateway/metallb | 先用 ClusterIP/NodePort，后续扩展 |
