import Personal from '@/views/system/Personal.vue'
import Admin from '@/views/system/Admin.vue'
import Role from '@/views/system/Role.vue'
import Dept from '@/views/system/Dept.vue'
import Post from '@/views/system/Post.vue'
import Menu from '@/views/system/Menu.vue'
import LoginLog from '@/views/monitor/LoginLog.vue'
import Operator from '@/views/monitor/Operator.vue'
import DBLog from '@/views/monitor/DBLog.vue'
import N9eConfig from '@/views/system/N9eConfig.vue'
import N9eMonitor from '@/views/monitor/N9eMonitor.vue'
import N9eDatasource from '@/views/monitor/N9eDatasource.vue'
import N9eOverview from '@/views/monitor/N9eOverview.vue'
import N9eSyncLog from '@/views/monitor/N9eSyncLog.vue'
import AlertRules from '@/views/monitor/AlertRules.vue'
import AlertEvents from '@/views/monitor/AlertEvents.vue'
const routes = [
    {
        path: '/system/personal',
        component: Personal,
        meta: {sTitle: '个人中心', tTitle: '个人信息'}
    },
    {
        path: '/system/admin',
        component: Admin,
        meta: {sTitle: '基础管理', tTitle: '用户信息'}
    },
    {
        path: '/system/role',
        component: Role,
        meta: {sTitle: '基础管理', tTitle: '角色信息'}
    },
    {
        path: '/system/menu',
        component: Menu,
        meta: {sTitle: '基础管理', tTitle: '菜单信息'}
    },
    {
        path: '/system/dept',
        component: Dept,
        meta: {sTitle: '基础管理', tTitle: '部门信息'}
    },
    {
        path: '/system/post',
        component: Post,
        meta: {sTitle: '基础管理', tTitle: '岗位信息'}
    },
    {
        path: '/monitor/loginlog',
        component: LoginLog,
        meta: {sTitle: '日志管理', tTitle: '登录日志'}
    },
    {
        path: '/monitor/operator',
        component: Operator,
        meta: {sTitle: '日志管理', tTitle: '操作日志'}
    },
        {
        path: '/monitor/dblog',
        component: DBLog,
        meta: {sTitle: '日志管理', tTitle: '数据日志'}
    },
    {
        path: '/system/n9e',
        component: N9eConfig,
        meta: {sTitle: '系统管理', tTitle: 'N9E 配置'}
    },
    {
        path: '/monitor/n9e',
        component: N9eMonitor,
        meta: {sTitle: '监控中心', tTitle: 'N9E 监控'}
    },
    {
        path: '/monitor/datasource',
        component: N9eDatasource,
        meta: {sTitle: '监控中心', tTitle: '数据源管理'}
    },
    {
        path: '/monitor/n9e-overview',
        component: N9eOverview,
        meta: {sTitle: '监控中心', tTitle: 'CMDB 总览'}
    },
    {
        path: '/monitor/sync-logs',
        component: N9eSyncLog,
        meta: {sTitle: '监控中心', tTitle: '同步日志'}
    },
    {
        path: '/monitor/alert-rules',
        component: AlertRules,
        meta: {sTitle: '监控中心', tTitle: '告警规则'}
    },
    {
        path: '/monitor/alert-events',
        component: AlertEvents,
        meta: {sTitle: '监控中心', tTitle: '告警事件'}
    }
]

export default routes

