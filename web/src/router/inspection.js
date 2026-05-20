const routes = [
    {
        path: '/inspection/overview',
        component: () => import('@/views/inspection/Overview.vue'),
        meta: { sTitle: '巡检管理', tTitle: '巡检概览' }
    },
    {
        path: '/inspection/tasks',
        component: () => import('@/views/inspection/Tasks.vue'),
        meta: { sTitle: '巡检管理', tTitle: '任务管理' }
    },
    {
        path: '/inspection/runs',
        component: () => import('@/views/inspection/Runs.vue'),
        meta: { sTitle: '巡检管理', tTitle: '运行历史' }
    },
    {
        path: '/inspection/runs/:id',
        component: () => import('@/views/inspection/RunDetail.vue'),
        meta: { sTitle: '巡检管理', tTitle: '运行详情', hidden: true }
    }
]

export default routes
