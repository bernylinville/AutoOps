import { createRouter, createWebHistory } from 'vue-router'
import systemRoutes from './system'
import cmdbRoutes from './cmdb'
import k8sRoutes from './k8s'
import configRoutes from './config'
import task from './task'
import tools from './Tools'
import app from './app'

// 路由集合
const routes = [
    {path: '/',      redirect: '/login'},
    {path: '/login', component: () => import('@/views/Login.vue')},
    {
        path: '/home',
        component: () => import('@/views/Home.vue'),
        redirect: '/dashboard',
        children: [
            {
                path: '/dashboard',
                component: () => import('@/views/dashboard/index.vue'),
                meta: {tTitle: '仪表盘'}
            },
            // 以下路由已注释，如需使用请先创建对应组件并导入
            // {
            //     path: '/dashboard/assets',
            //     component: AssetsManagement,
            //     meta: {tTitle: '资产管理'}
            // },
            // {
            //     path: '/dashboard/services',
            //     component: ServiceManagement,
            //     meta: {tTitle: '服务管理'}
            // },
            // {
            //     path: '/dashboard/monitoring',
            //     component: MonitoringAlerts,
            //     meta: {tTitle: '监控告警'}
            // },
            // {
            //     path: '/dashboard/users',
            //     component: UserStatistics,
            //     meta: {tTitle: '用户统计'}
            // },
            ...systemRoutes,
            ...cmdbRoutes,
            ...k8sRoutes,
            ...configRoutes,
            ...task,
            ...tools,
            ...app,
        ]
    }
]

// 创建 router 实例 ✅ 唯一的一次
const router = createRouter({
    history: createWebHistory(),
    routes // 使用上面定义的 routes 数组
})

// 引入 storage 工具类
import storage from '@/utils/storage'

// 挂载导航守卫
router.beforeEach((to, from, next) => {
    const token = storage.getItem('token')
    const sysAdmin = storage.getItem('sysAdmin')

    // 如果已经登录，且访问的是登录页，则跳转到首页
    if (to.path === '/login' && token && sysAdmin) {
        return next('/dashboard')
    }

    // 如果访问的是非登录页，检查是否登录
    if (to.path !== '/login') {
        if (!token || !sysAdmin) {
            // 未登录，跳转到登录页
            return next('/login')
        }
    }

    // 其他情况正常放行
    next()
})

export default router
