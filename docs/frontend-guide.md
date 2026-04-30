# Frontend Development Guide

> 前端开发时阅读：新增页面、修改 API 调用、调整路由和状态管理。

## 项目结构

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

## API 层

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

- URL **不需要**写 `/api/v1` 前缀 — `request.js` 拦截器自动添加
- `params` 用于 GET 查询参数，`data` 用于 POST/PUT/DELETE 请求体
- 所有方法返回 Axios Promise

**Axios 实例配置**（`src/utils/request.js`）：
- baseURL: 空字符串（依赖 devServer proxy 或 nginx 代理）
- timeout: 15000ms
- 请求拦截器：自动注入 `Authorization: Bearer <token>`
- 响应拦截器：`code === 401/406` 时自动清除 localStorage 并跳转 `/login`

## 路由

**组织方式**：每个模块一个路由文件（`router/{module}.js`），导出路由数组，在 `router/router.js` 中 spread 到 `/home` 的 children。

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

## 状态管理

Vuex store 5 个 key，全部持久化到 localStorage：

| Key | 用途 | 来源 |
|-----|------|------|
| `token` | JWT Token | 登录接口返回 |
| `sysAdmin` | 当前用户信息 | 登录接口返回 |
| `leftMenuList` | 左侧菜单树 | 登录接口返回 |
| `permissionList` | 权限码列表 | 登录接口返回 |
| `activePath` | 当前激活菜单路径 | 导航时更新 |

## 环境配置

### localStorage 命名空间

`src/utils/storage.js` 使用 `process.env.VUE_APP_NAME_SPACE` 作为 localStorage key。

**已知问题**：`web/src/.env` 定义了 `VUE_APP_NAME_SPACE: 'devops-api'`，但 Vue CLI 构建时**未注入此变量**（文件在 `src/` 而非项目根目录，且使用 YAML 冒号语法而非 dotenv `=` 语法）。实际构建后 localStorage key 为字符串 `"undefined"`。

### devServer Proxy

`vue.config.js` 中 proxy 配置：

```javascript
proxy: {
  '/api/v1': {
    target: 'http://192.168.1.156:5700',  // ⚠️ 硬编码，本地开发需修改
    changeOrigin: true
  }
}
```

本地开发时需改为指向你的后端地址（如 `http://localhost:8000`）。

使用 Docker 全栈开发时不需要 proxy — nginx 直接代理。

## 新增页面 Checklist

1. 创建 `src/api/{module}.js` — 添加 API 方法
2. 创建 `src/views/{module}/{Page}.vue` — 页面组件
3. 在 `src/router/{module}.js` — 添加路由配置
4. 在后端 seed 菜单数据（`sys_menu` 表）— 控制左侧菜单显示
5. 分配 RBAC 权限码 — 控制按钮级权限
6. UI 样式参考 [docs/design-system.md](design-system.md)

## 部署中心页面

当前已新增基础页面：

- `src/views/K8s/K8sReleaseCenter.vue`

当前已接入基础能力：

- 部署申请列表
- 部署目标列表
- 新建部署申请弹窗
- 审批状态同步
- 审批重发
- 执行记录查看
- 直连凭据校验
- GitOps 工作树 / 仓库校验

对应 API：

- `src/api/deploy.js`

说明：

- 当前页面是基础运维页，不是最终交互稿
- 后续如果继续完善，应优先补详情侧栏、审批时间线、申请表单校验，以及 Direct / GitOps 差异化字段展示
