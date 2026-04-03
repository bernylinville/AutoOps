import request from "@/utils/request"

export default {
    // 分页项目列表
    getProjectList(params) {
        return request({ url: 'cmdb/project/list', method: 'get', params })
    },
    // 项目详情
    getProjectDetail(id) {
        return request({ url: 'cmdb/project/detail', method: 'get', params: { id } })
    },
    // 创建项目
    createProject(data) {
        return request({ url: 'cmdb/project', method: 'post', data })
    },
    // 更新项目
    updateProject(data) {
        return request({ url: 'cmdb/project', method: 'put', data })
    },
    // 删除项目
    deleteProject(id) {
        return request({ url: 'cmdb/project', method: 'delete', params: { id } })
    },
    // 项目资产统计
    getProjectStats(id) {
        return request({ url: 'cmdb/project/stats', method: 'get', params: { id } })
    },
    // 项目关联主机（分页）
    getProjectHosts(params) {
        return request({ url: 'cmdb/project/hosts', method: 'get', params })
    },
    // 项目关联数据库（分页）
    getProjectDatabases(params) {
        return request({ url: 'cmdb/project/databases', method: 'get', params })
    },
    // 项目关联应用
    getProjectApps(projectId) {
        return request({ url: 'cmdb/project/apps', method: 'get', params: { projectId } })
    }
}
