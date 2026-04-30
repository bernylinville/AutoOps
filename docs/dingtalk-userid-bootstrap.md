# DingTalk UserId 补录指南

AutoOps 发起钉钉 OA 审批实例时，`originatorUserId` 和 `approvers[].userIds` 必须是企业通讯录里的真实钉钉 `userId`。不能使用手机号、姓名、unionId，也不能使用 `smoke-dingtalk-admin` 这类占位值。

## 1. 查询真实 `userId`

推荐方式：

1. 进入钉钉管理后台或开放平台通讯录相关页面。
2. 找到用于测试的发起人和审批人。
3. 记录该成员的企业内 `userId`。

如果应用已开通通讯录读权限，也可以通过接口查询成员列表。当前 MVP 不强依赖自动查询，优先手工补录。

## 2. 补录到 AutoOps

先检查当前数据：

```bash
docker exec devops-postgres psql -U devops -d autoops -c "select id, username, nickname, dingtalk_user_id from sys_admin order by id;"
```

补录示例：

```bash
docker exec devops-postgres psql -U devops -d autoops -c "UPDATE sys_admin SET dingtalk_user_id='<真实钉钉 userId>' WHERE id=89;"
```

如果发起人和审批人不是同一个 AutoOps 用户，需要分别补录。

## 3. 验证补录结果

```bash
docker exec devops-postgres psql -U devops -d autoops -c "select id, username, nickname, dingtalk_user_id from sys_admin where id=89;"
```

期望：

- `dingtalk_user_id` 不为空
- 值不是 `smoke-*` 或其他占位字符串
- 该成员在钉钉企业内未离职、未禁用

## 4. 重新发起审批实例

补录后可以重新创建部署申请，或对待审批申请调用重发审批接口：

```bash
curl -sS -X POST "http://127.0.0.1:18000/api/v1/integrations/agent/deploy-requests/<requestNo>/dispatch-approval" \
  -H "Authorization: Bearer <agent token>" \
  -H "Content-Type: application/json"
```

成功标准：

- `approvalDispatchStatus=dispatched`
- `dingtalkProcessInstanceId` 不为空
- 钉钉 OA 中可见审批单

