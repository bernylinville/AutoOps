import request from "@/utils/request"

export default {
    // ========================================
    // CI 类型管理
    // ========================================
    getCITypeList() {
        return request({
            url: 'cmdb/ci/type/list',
            method: 'get'
        })
    },
    getCITypeDetail(id) {
        return request({
            url: 'cmdb/ci/type/detail',
            method: 'get',
            params: { id }
        })
    },
    createCIType(data) {
        return request({
            url: 'cmdb/ci/type',
            method: 'post',
            data
        })
    },
    updateCIType(data) {
        return request({
            url: 'cmdb/ci/type',
            method: 'put',
            data
        })
    },
    deleteCIType(id) {
        return request({
            url: 'cmdb/ci/type',
            method: 'delete',
            params: { id }
        })
    },

    // ========================================
    // CI 属性管理
    // ========================================
    getCITypeAttributes(ciTypeId) {
        return request({
            url: 'cmdb/ci/attribute/list',
            method: 'get',
            params: { ciTypeId }
        })
    },
    createCITypeAttribute(data) {
        return request({
            url: 'cmdb/ci/attribute',
            method: 'post',
            data
        })
    },
    updateCITypeAttribute(data) {
        return request({
            url: 'cmdb/ci/attribute',
            method: 'put',
            data
        })
    },
    deleteCITypeAttribute(id) {
        return request({
            url: 'cmdb/ci/attribute',
            method: 'delete',
            params: { id }
        })
    },

    // ========================================
    // CI 实例管理
    // ========================================
    getCIInstanceList(params) {
        return request({
            url: 'cmdb/ci/instance/list',
            method: 'get',
            params
        })
    },
    getCIInstanceDetail(id) {
        return request({
            url: 'cmdb/ci/instance/detail',
            method: 'get',
            params: { id }
        })
    },
    createCIInstance(data) {
        return request({
            url: 'cmdb/ci/instance',
            method: 'post',
            data
        })
    },
    updateCIInstance(data) {
        return request({
            url: 'cmdb/ci/instance',
            method: 'put',
            data
        })
    },
    deleteCIInstance(id) {
        return request({
            url: 'cmdb/ci/instance',
            method: 'delete',
            params: { id }
        })
    },

    // ========================================
    // CI 关系管理
    // ========================================
    getCIRelations(ciId) {
        return request({
            url: 'cmdb/ci/relation/list',
            method: 'get',
            params: { ciId }
        })
    },
    createCIRelation(data) {
        return request({
            url: 'cmdb/ci/relation',
            method: 'post',
            data
        })
    },
    deleteCIRelation(id) {
        return request({
            url: 'cmdb/ci/relation',
            method: 'delete',
            params: { id }
        })
    },

    // ========================================
    // CI 拓扑图（Phase 3）
    // ========================================
    getAllCIInstances(keyword = '') {
        return request({
            url: 'cmdb/ci/instance/all',
            method: 'get',
            params: { keyword }
        })
    },
    getCITopology(ciId, direction = 'all') {
        return request({
            url: 'cmdb/ci/topology',
            method: 'get',
            params: { ciId, direction }
        })
    }
}
