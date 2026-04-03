const routes = [
    {
        path: '/system/personal',
        component: () => import('@/views/system/Personal.vue'),
        meta: {sTitle: '个人中心', tTitle: '个人信息'}
    },
    {
        path: '/system/admin',
        component: () => import('@/views/system/Admin.vue'),
        meta: {sTitle: '基础管理', tTitle: '用户信息'}
    },
    {
        path: '/system/role',
        component: () => import('@/views/system/Role.vue'),
        meta: {sTitle: '基础管理', tTitle: '角色信息'}
    },
    {
        path: '/system/menu',
        component: () => import('@/views/system/Menu.vue'),
        meta: {sTitle: '基础管理', tTitle: '菜单信息'}
    },
    {
        path: '/system/dept',
        component: () => import('@/views/system/Dept.vue'),
        meta: {sTitle: '基础管理', tTitle: '部门信息'}
    },
    {
        path: '/system/post',
        component: () => import('@/views/system/Post.vue'),
        meta: {sTitle: '基础管理', tTitle: '岗位信息'}
    },
    {
        path: '/monitor/loginlog',
        component: () => import('@/views/monitor/LoginLog.vue'),
        meta: {sTitle: '日志管理', tTitle: '登录日志'}
    },
    {
        path: '/monitor/operator',
        component: () => import('@/views/monitor/Operator.vue'),
        meta: {sTitle: '日志管理', tTitle: '操作日志'}
    },
        {
        path: '/monitor/dblog',
        component: () => import('@/views/monitor/DBLog.vue'),
        meta: {sTitle: '日志管理', tTitle: '数据日志'}
    },
    {
        path: '/monitor/audit-log',
        component: () => import('@/views/monitor/AuditLog.vue'),
        meta: {sTitle: '日志管理', tTitle: '审计日志'}
    },
    {
        path: '/system/n9e',
        component: () => import('@/views/system/N9eConfig.vue'),
        meta: {sTitle: '系统管理', tTitle: 'N9E 配置'}
    },
    {
        path: '/monitor/n9e',
        component: () => import('@/views/monitor/N9eMonitor.vue'),
        meta: {sTitle: '监控中心', tTitle: 'N9E 监控'}
    },
    {
        path: '/monitor/datasource',
        component: () => import('@/views/monitor/N9eDatasource.vue'),
        meta: {sTitle: '监控中心', tTitle: '数据源管理'}
    },
    {
        path: '/monitor/n9e-overview',
        component: () => import('@/views/monitor/N9eOverview.vue'),
        meta: {sTitle: '监控中心', tTitle: 'CMDB 总览'}
    },
    {
        path: '/monitor/sync-logs',
        component: () => import('@/views/monitor/N9eSyncLog.vue'),
        meta: {sTitle: '监控中心', tTitle: '同步日志'}
    },
    {
        path: '/monitor/alert-rules',
        component: () => import('@/views/monitor/AlertRules.vue'),
        meta: {sTitle: '监控中心', tTitle: '告警规则'}
    },
    {
        path: '/monitor/alert-events',
        component: () => import('@/views/monitor/AlertEvents.vue'),
        meta: {sTitle: '监控中心', tTitle: '告警事件'}
    }
]

export default routes
