const routes = [
    {
        path: '/config/ecskey',
        component: () => import('@/views/configcenter/ecs-key.vue'),
        meta: {sTitle: '配置中心', tTitle: '主机凭据'}
    },
    {
        path: '/config/accountauth',
        component: () => import('@/views/configcenter/accountauth.vue'),
        meta: {sTitle: '配置中心', tTitle: '通用凭据'}
    },
    {
        path: '/config/keymanage',
        component: () => import('@/views/configcenter/KeyManage'),
        meta: {sTitle: '配置中心', tTitle: '密钥管理'}
    }
]

export default routes
