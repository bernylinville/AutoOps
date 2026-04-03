const routes = [
    {
        path: '/k8s/list',
        component: () => import('@/views/K8s/k8s-clusters.vue'),
        meta: {sTitle: '容器管理', tTitle: '集群管理'}
    },
    {
        path: '/k8s/cluster/:clusterId',
        component: () => import('@/views/K8s/clusters/K8sDetails.vue'),
        props: true,
        meta: {sTitle: '容器管理', tTitle: '集群详情'}
    },
    {
        path: '/k8s/cluster/:clusterId/node/:nodeName',
        component: () => import('@/views/K8s/nodes/NodeDetails.vue'),
        props: true,
        meta: {sTitle: '容器管理', tTitle: '节点详情'}
    },
    {
        path: '/k8s/node',
        component: () => import('@/views/K8s/k8s-nodes.vue'),
        meta: {sTitle: '容器管理', tTitle: '节点管理'}
    },
    {
        path: '/k8s/namespace',
        component: () => import('@/views/K8s/k8s-namespace.vue'),
        meta: {sTitle: '容器管理', tTitle: '命名空间'}
    },
    {
        path: '/k8s/workload',
        component: () => import('@/views/K8s/k8s-workloads.vue'),
        meta: {sTitle: '容器管理', tTitle: '工作负载'}
    },
        {
        path: '/k8s/network',
        component: () => import('@/views/K8s/k8s-network.vue'),
        meta: {sTitle: '容器管理', tTitle: '网络管理'}
    },
        {
        path: '/k8s/config',
        component: () => import('@/views/K8s/k8s-config.vue'),
        meta: {sTitle: '容器管理', tTitle: '配置管理'}
    },
        {
        path: '/k8s/storage',
        component: () => import('@/views/K8s/k8s-storage'),
        meta: {sTitle: '容器管理', tTitle: '存储管理'}
    },
    {
        path: '/k8s/pod/:clusterId/:namespace/:podName',
        component: () => import('@/views/K8s/pods/k8s-pod.vue'),
        props: true,
        meta: {sTitle: '容器管理', tTitle: '容器详情'}
    },
    {
        path: '/k8s/terminal/:clusterId/:namespace/:podName',
        component: () => import('@/views/K8s/pods/K8S-sterminal.vue'),
        props: true,
        meta: {sTitle: '容器管理', tTitle: '容器终端'}
    },
    {
        path: '/k8s/monitoring',
        component: () => import('@/views/K8s/nodes/k8s-monitoring.vue'),
        meta: {sTitle: '容器管理', tTitle: '监控仪表板'}
    },
]

export default routes
