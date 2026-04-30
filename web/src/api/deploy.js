import request from '@/utils/request'

export default {
  getClusterTargets() {
    return request({
      url: '/deploy/cluster-targets',
      method: 'get'
    })
  },

  getClusterTargetById(id) {
    return request({
      url: `/deploy/cluster-targets/${id}`,
      method: 'get'
    })
  },

  createClusterTarget(data) {
    return request({
      url: '/deploy/cluster-targets',
      method: 'post',
      data
    })
  },

  updateClusterTarget(id, data) {
    return request({
      url: `/deploy/cluster-targets/${id}`,
      method: 'put',
      data
    })
  },

  getDeployRequests(params) {
    return request({
      url: '/deploy/requests',
      method: 'get',
      params
    })
  },

  createDeployRequest(data) {
    return request({
      url: '/deploy/requests',
      method: 'post',
      data
    })
  },

  executeDeployRequest(id, data = {}) {
    return request({
      url: `/deploy/requests/${id}/execute`,
      method: 'post',
      data
    })
  },

  retryApprovalDispatch(id) {
    return request({
      url: `/deploy/requests/${id}/dispatch-approval`,
      method: 'post'
    })
  },

  syncApprovalStatus(id) {
    return request({
      url: `/deploy/requests/${id}/sync-approval`,
      method: 'post'
    })
  },

  validateDirectCredential(id) {
    return request({
      url: `/deploy/cluster-targets/${id}/validate-direct-credential`,
      method: 'post'
    })
  },

  validateGitOpsRepo(id) {
    return request({
      url: `/deploy/cluster-targets/${id}/validate-gitops-repo`,
      method: 'post'
    })
  },

  validateGitOpsWorkingTree() {
    return request({
      url: '/deploy/gitops/validate-working-tree',
      method: 'get'
    })
  },

  getExecutionRecords(id) {
    return request({
      url: `/deploy/requests/${id}/executions`,
      method: 'get'
    })
  },

  getDeployNotifications(id) {
    return request({
      url: `/deploy/requests/${id}/notifications`,
      method: 'get'
    })
  },

  rollbackDeployRequest(id) {
    return request({
      url: `/deploy/requests/${id}/rollback`,
      method: 'post'
    })
  }
}
