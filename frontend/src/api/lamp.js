import request from './request'

// 路灯台账接口。
export const lampApi = {
  list: (params) => request.get('/lamps', { params }),
  detail: (id) => request.get(`/lamps/${id}`),
  create: (data) => request.post('/lamps', data),
  update: (id, data) => request.put(`/lamps/${id}`, data),
  remove: (id) => request.delete(`/lamps/${id}`),
  options: () => request.get('/lamps/options'),
  statistics: () => request.get('/lamps/statistics'),
}
