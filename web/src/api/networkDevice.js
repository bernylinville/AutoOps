import request from "@/utils/request"

export default {
    getNetworkDevices(params) {
        return request({ url: 'cmdb/network/list', method: 'get', params })
    },
    inspectDevice(ciInstanceId) {
        return request({ url: 'cmdb/network/inspect', method: 'post', data: { ciInstanceId } })
    },
    getInspectionHistory(ciId, page = 1, pageSize = 20) {
        return request({ url: 'cmdb/network/inspect/history', method: 'get', params: { ciId, page, pageSize } })
    }
}
