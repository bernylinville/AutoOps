const routes = [
    {
        path: '/cmdb/ecs',
        component: () => import('@/views/cmdb/cmdbHost.vue'),
        meta: {sTitle: '资产管理', tTitle: '主机管理'}
    },
    {
        path: '/cmdb/group',
        component: () => import('@/views/cmdb/cmdbGroup.vue'),
        meta: {sTitle: '资产管理', tTitle: '业务分组'}
    },
    {
        path: '/cmdb/db',
        component: () => import('@/views/cmdb/cmdbDB.vue'),
        meta: {sTitle: '资产管理', tTitle: '数据管理'}
    },
    {
        path: '/cmdb/ssh',
        component: () => import('@/views/cmdb/Host/SSH.vue'),
        meta: {sTitle: '资产管理', tTitle: '终端登录'}
    },
    {
        path: '/cmdb/dbdetails',
        component: () => import('@/views/cmdb/DBdetails.vue'),
        meta: {sTitle: '数据管理', tTitle: '数据库操作'}
    },
    {
        path: '/cmdb/switch',
        component: () => import('@/views/cmdb/cmdbSwitch.vue'),
        meta: {sTitle: '资产管理', tTitle: '网络设备'}
    },
    {
        path: '/cmdb/ci',
        component: () => import('@/views/cmdb/CIManage.vue'),
        meta: {sTitle: '资产管理', tTitle: 'CI模型管理'}
    },
    {
        path: '/cmdb/ci/topology',
        component: () => import('@/views/cmdb/CITopology.vue'),
        meta: {sTitle: '资产管理', tTitle: 'CI拓扑图'}
    },
    {
        path: '/cmdb/changelog',
        component: () => import('@/views/cmdb/CIChangeLog.vue'),
        meta: {sTitle: '资产管理', tTitle: '变更日志'}
    },
    {
        path: '/cmdb/network',
        component: () => import('@/views/cmdb/NetworkDevice.vue'),
        meta: {sTitle: '资产管理', tTitle: '网络设备管理'}
    },
    {
        path: '/cmdb/project',
        component: () => import('@/views/cmdb/ProjectList.vue'),
        meta: {sTitle: '资产管理', tTitle: '项目管理'}
    },
    {
        path: '/cmdb/project/detail/:id',
        component: () => import('@/views/cmdb/ProjectDetail.vue'),
        meta: {sTitle: '资产管理', tTitle: '项目详情'}
    }

]

export default routes
