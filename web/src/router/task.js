const routes = [
    {
        path: '/task/template',
        component: () => import('@/views/task/TaskTemplate.vue'),
        meta: { sTitle: '任务中心', tTitle: '任务模版' }
    },
    {
        path: '/task/job',
        component: () => import('@/views/task/TaskJob.vue'),
        meta: { sTitle: '任务中心', tTitle: '任务作业' }
    },
    {
        path: '/task/ansible',
        component: () => import('@/views/task/TaskAnsible.vue'),
        meta: { sTitle: '任务中心', tTitle: 'Ansible任务' }
    },
    {
        path: '/task/config',
        component: () => import('@/views/task/TaskConfig.vue'),
        meta: { sTitle: '任务中心', tTitle: '配置管理' }
    },
    {
        path: '/task/ansible/history',
        name: 'AnsibleTaskHistory',
        component: () => import('@/views/task/AnsibleTaskHistory.vue'),
        meta: { sTitle: '任务中心', tTitle: '执行历史', hidden: true }
    }
]

export default routes
