# Frontend Development Guide

前端开发时阅读：新增页面、修改 API 调用、调整路由和状态管理。

## 1. 项目结构

```
web/src/
  api/              # Axios API 模块（含 deploy.js）
  assets/           # 静态资源
  components/       # 公共组件
  permission/       # v-authority 权限指令
  router/           # 按模块拆分的路由（system, cmdb, k8s, config, task, tools, app）
  store/            # Vuex（index.js + mutations.js）
  utils/            # request.js（Axios 实例）、storage.js（localStorage 封装）
  views/            # 按模块组织的页面（dashboard, system, cmdb, k8s, configcenter, monitor, work, app）
```

## 2. API 层

**封装模式**（`src/api/{module}.js`）：

```javascript
import request from "@/utils/request"
export default {
  getHostList(params) {
    return request({ url: 'cmdb/host/list', method: 'get', params })
  },
  createHost(data) {
    return request({ url: 'cmdb/host/create', method: 'post', data })
  }
}
```

关键规则：

- URL 不写 `/api/v1` 前缀 —— `request.js` 拦截器自动添加
- `params` 用于 GET 查询参数，`data` 用于 POST / PUT / DELETE 请求体
- 所有方法返回 Axios Promise

**Axios 实例配置**（`src/utils/request.js`）：

- `baseURL`：空字符串（依赖 devServer proxy 或 Nginx 反代）
- `timeout`：15000 ms
- 请求拦截器：自动注入 `Authorization: Bearer <token>`
- 响应拦截器：`code === 401` 或 `406` 时自动清除 localStorage 并跳转 `/login`

## 3. 路由

每个模块一个路由文件（`router/{module}.js`），导出路由数组，在 `router/router.js` 中展开到 `/home` 的 `children`。

**路由守卫**（`router/router.js`）：

```javascript
router.beforeEach((to, from, next) => {
  const token = storage.getItem('token')
  const sysAdmin = storage.getItem('sysAdmin')
  if (to.path === '/login' && token && sysAdmin) return next('/dashboard')
  if (to.path !== '/login' && (!token || !sysAdmin)) return next('/login')
  next()
})
```

## 4. 状态管理

Vuex store 共 5 个 key，全部持久化到 localStorage：

| Key | 用途 | 来源 |
|-----|------|------|
| `token` | JWT Token | 登录接口返回 |
| `sysAdmin` | 当前用户信息 | 登录接口返回 |
| `leftMenuList` | 左侧菜单树 | 登录接口返回 |
| `permissionList` | 权限码列表 | 登录接口返回 |
| `activePath` | 当前激活菜单路径 | 导航时更新 |

## 5. 环境配置

### localStorage 命名空间

`src/utils/storage.js` 使用 `process.env.VUE_APP_NAME_SPACE` 作为 localStorage key。

已知问题：`web/src/.env` 中定义了 `VUE_APP_NAME_SPACE: 'devops-api'`，但 Vue CLI 构建时未注入此变量（文件在 `src/` 而非项目根目录，使用 YAML 冒号语法而非 dotenv `=` 语法）。实际构建后 localStorage key 为字符串 `"undefined"`。

### devServer Proxy

`vue.config.js` 中 proxy 配置的 target 硬编码为 `http://192.168.1.156:5700`，本地开发需改为本机后端地址（如 `http://localhost:8000`）。

Docker 全栈开发时不需要 proxy —— Nginx 直接反代 API。

## 6. 新增页面 Checklist

1. 创建 `src/api/{module}.js` —— 添加 API 方法
2. 创建 `src/views/{module}/{Page}.vue` —— 页面组件
3. 在 `src/router/{module}.js` 中添加路由配置
4. 在后端 seed 菜单数据（`sys_menu` 表）中控制左侧菜单显示
5. 分配 RBAC 权限码，控制按钮级权限
6. UI 样式参考 [docs/design-system.md](design-system.md)

## 7. 部署中心页面

当前能力：

- 部署申请列表
- 部署目标列表
- 新建部署申请弹窗
- 审批状态同步与重发
- 执行记录查看
- 直连凭据校验
- GitOps 工作树 / 仓库校验

对应 API：`src/api/deploy.js`

后续完善方向：详情侧栏、审批时间线、申请表单校验、Direct / GitOps 差异化字段展示。
