import request from "@/utils/request"

export default {
    getChangeLogs(params) {
        return request({
            url: 'cmdb/changelog/list',
            method: 'get',
            params
        })
    }
}
