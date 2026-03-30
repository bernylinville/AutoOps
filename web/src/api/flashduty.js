import request from "@/utils/request"

export default {
    // ========= 配置 =========

    // 获取 FlashDuty 配置状态
    getConfigStatus() {
        return request({
            url: 'flashduty/config/status',
            method: 'get'
        })
    },

    // 测试 FlashDuty 连接
    testConnection() {
        return request({
            url: 'flashduty/test-connection',
            method: 'post'
        })
    },

    // ========= 告警 =========

    // 获取活跃告警列表
    getActiveAlerts(params = {}) {
        return request({
            url: 'flashduty/alerts/active',
            method: 'get',
            params
        })
    },

    // 获取告警概况统计
    getAlertSummary() {
        return request({
            url: 'flashduty/alerts/summary',
            method: 'get'
        })
    },

    // 获取指定主机的告警
    getAlertsByHost(ident, params = {}) {
        return request({
            url: `flashduty/alerts/host/${ident}`,
            method: 'get',
            params
        })
    },

    // ========= 故障 =========

    // 获取活跃故障列表
    getActiveIncidents(params = {}) {
        return request({
            url: 'flashduty/incidents/active',
            method: 'get',
            params
        })
    },

    // 认领故障
    claimIncident(id) {
        return request({
            url: `flashduty/incidents/${id}/claim`,
            method: 'post'
        })
    },

    // 关闭故障
    closeIncident(id, desc = '') {
        return request({
            url: `flashduty/incidents/${id}/close`,
            method: 'post',
            data: { desc }
        })
    },

    // ========= 值班 =========

    // 获取值班表列表
    getSchedules(params = {}) {
        return request({
            url: 'flashduty/schedules',
            method: 'get',
            params
        })
    },

    // 获取今日值班人
    getTodayOnCall(params = {}) {
        return request({
            url: 'flashduty/oncall/today',
            method: 'get',
            params
        })
    },

    // ========= SRE 指标 =========

    // 获取 SRE 指标 (MTTA/MTTR/降噪率)
    getSREMetrics(days = 7) {
        return request({
            url: 'flashduty/insight/metrics',
            method: 'get',
            params: { days }
        })
    },

    // 获取趋势数据
    getTrendData(days = 7) {
        return request({
            url: 'flashduty/insight/trend',
            method: 'get',
            params: { days }
        })
    }
}
