import request from "@/utils/request"

export default {
    // --- Tasks ---
    getTasks(params) {
        return request({ url: 'inspection/tasks', method: 'get', params })
    },
    getTask(id) {
        return request({ url: `inspection/tasks/${id}`, method: 'get' })
    },
    updateTask(id, data) {
        return request({ url: `inspection/tasks/${id}`, method: 'put', data })
    },
    triggerTask(id) {
        return request({ url: `inspection/tasks/${id}/trigger`, method: 'post' })
    },

    // --- Runs ---
    getRuns(params) {
        return request({ url: 'inspection/runs', method: 'get', params })
    },
    getRun(id) {
        return request({ url: `inspection/runs/${id}`, method: 'get' })
    },
    getRunResults(runId, params) {
        return request({ url: `inspection/runs/${runId}/results`, method: 'get', params })
    },
    getRunAlerts(runId, params) {
        return request({ url: `inspection/runs/${runId}/alerts`, method: 'get', params })
    },

    // --- Overview ---
    getOverview() {
        return request({ url: 'inspection/overview', method: 'get' })
    },

    // --- Report Download ---
    downloadReport(runId) {
        return request({
            url: `inspection/runs/${runId}/report`,
            method: 'get',
            responseType: 'blob'
        })
    }
}
