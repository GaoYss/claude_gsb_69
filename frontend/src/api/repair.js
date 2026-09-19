import request from './request'

// 维修记录接口。
export const repairApi = {
  list: (params) => request.get('/repairs', { params }),
  detail: (id) => request.get(`/repairs/${id}`),
  listByFault: (faultId) => request.get(`/repairs/fault/${faultId}`),
  create: (data) => request.post('/repairs', data),
  update: (id, data) => request.put(`/repairs/${id}`, data),
  finish: (id, data) => request.post(`/repairs/${id}/finish`, data),
  remove: (id) => request.delete(`/repairs/${id}`),
  meta: () => request.get('/repairs/meta'),
  statistics: () => request.get('/repairs/statistics'),
}
