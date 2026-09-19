import request from './request'

// 维修状态查询接口。
export const statusApi = {
  overview: () => request.get('/status/overview'),
  lamps: (params) => request.get('/status/lamps', { params }),
  track: (params) => request.get('/status/track', { params }),
}
