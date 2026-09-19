import request from './request'

// 故障登记接口。
export const faultApi = {
  list: (params) => request.get('/faults', { params }),
  detail: (id) => request.get(`/faults/${id}`),
  create: (data) => request.post('/faults', data),
  update: (id, data) => request.put(`/faults/${id}`, data),
  remove: (id) => request.delete(`/faults/${id}`),
  close: (id, data) => request.post(`/faults/${id}/close`, data),
  meta: () => request.get('/faults/meta'),
}
