const routes = [
    {
        path: '/app/application',
        component: () => import('@/views/app/application.vue'),
        meta: {sTitle: '服务管理', tTitle: '应用列表'}
    },
    {
        path: '/app/quick-release',
        component: () => import('@/views/app/app_quick_release.vue'),
        meta: {sTitle: '服务管理', tTitle: '快速发布'}
    },
    {
        path: '/app/quick-temp/:id',
        component: () => import('@/views/app/app_quick_temp.vue'),
        meta: {sTitle: '服务管理', tTitle: '发布模板'}
    },
]

export default routes
