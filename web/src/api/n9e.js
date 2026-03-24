import request from "@/utils/request"

export default {
    // 获取 N9E 配置
    getConfig() {
        return request({
            url: 'n9e/config',
            method: 'get'
        })
    },

    // 保存 N9E 配置
    saveConfig(data) {
        return request({
            url: 'n9e/config',
            method: 'post',
            data
        })
    },

    // 测试 N9E 连接
    testConnection(data) {
        return request({
            url: 'n9e/test-connection',
            method: 'post',
            data
        })
    },

    // 触发数据同步
    triggerSync() {
        return request({
            url: 'n9e/sync',
            method: 'post'
        })
    },

    // 获取同步状态
    getSyncStatus() {
        return request({
            url: 'n9e/sync/status',
            method: 'get'
        })
    },

    // 获取业务组列表
    getBusiGroups() {
        return request({
            url: 'n9e/busi-groups',
            method: 'get'
        })
    },

    // 获取数据源列表
    getDatasources() {
        return request({
            url: 'n9e/datasources',
            method: 'get'
        })
    },

    // PromQL 查询
    queryPromQL(params) {
        return request({
            url: 'n9e/query',
            method: 'get',
            params
        })
    },

    // 获取总览统计
    getOverview() {
        return request({
            url: 'n9e/overview',
            method: 'get'
        })
    },

    // 获取同步日志
    getSyncLogs(limit = 20) {
        return request({
            url: 'n9e/sync/logs',
            method: 'get',
            params: { limit }
        })
    },

    // 检测数据源连通性
    checkDatasource(id) {
        return request({
            url: `n9e/datasources/${id}/check`,
            method: 'post'
        })
    }
}
