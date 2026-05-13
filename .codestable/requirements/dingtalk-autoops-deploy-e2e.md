---
doc_type: requirement
slug: dingtalk-autoops-deploy-e2e
pitch: 在钉钉群里说清楚要部署哪个仓库，系统完成审批、构建、发布和结果回群
status: current
last_reviewed: 2026-05-13
implemented_by:
  - deploy-dingtalk-autoops-e2e
tags: [dingtalk, deploy, e2e]
---

# 钉钉群发起测试环境部署

## 用户故事

- 作为在群里提出发布需求的业务研发，希望用一句话说明 Git 仓库、分支和测试环境，而不是手动填写镜像、namespace 和 Jenkins job。
- 作为审批人，希望发布先进入 OA 审批，通过后系统再构建和部署，而不是机器人绕过审批直接上线。
- 作为值班或测试人员，希望部署完成后在群里看到状态、镜像、namespace 和访问地址，而不是分别打开 Jenkins、Harbor 和 Kubernetes 排查。
- 作为平台维护者，希望机器人只负责理解需求，真正的审批、构建、部署和通知都由 AutoOps 记录状态和执行。

## 为什么需要

测试环境发布常常从群聊发起，但实际操作分散在机器人、审批、Jenkins、镜像仓库和 Kubernetes 中。人工串联时容易遗漏审批、填错部署参数，也很难在失败后说明卡在哪一步。需要一条可追踪的发布流程，让自然语言入口、审批、构建、部署和结果通知都能被同一个系统记录。

## 怎么解决

用户在钉钉群里描述要部署的仓库、分支和环境后，机器人把请求交给 AutoOps。AutoOps 创建部署申请并发起审批；审批通过后自动构建镜像、发布到测试环境，并把最终状态和访问地址发回群里。失败时，结果中保留失败阶段和错误信息，便于继续处理。

## 边界

- 当前能力只覆盖测试环境的最小验证路径，不表示生产环境、多集群复杂审批或正式发布已经可用。
- 用户不提交镜像、namespace、ClusterTarget、Jenkins job 或 Harbor 仓库；这些由 AutoOps 的应用配置和环境配置决定。
- 机器人不直接操作 Jenkins、Harbor、Kubernetes 或 GitOps 仓库。
- 对外访问当前使用 NodePort 的节点 IP 地址，不提供 Gateway / Ingress 固定域名能力。
- 该能力不负责定义所有语言的构建模板；业务仓库仍需要自己的 Jenkinsfile 按约定输出镜像结果。

## 变更日志

- 2026-05-13：基于 `java-demo` 钉钉最小 E2E 验收结果回填为当前能力。
