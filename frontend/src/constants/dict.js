// 路灯运行状态。
export const RUN_STATUS = {
  normal: { label: '正常', type: 'success' },
  fault: { label: '故障', type: 'danger' },
  maintenance: { label: '维修中', type: 'warning' },
  offline: { label: '停用', type: 'info' },
}

// 故障处理状态。
export const FAULT_STATUS = {
  pending: { label: '待处理', type: 'danger' },
  processing: { label: '维修中', type: 'warning' },
  repaired: { label: '已修复', type: 'success' },
  closed: { label: '已关闭', type: 'info' },
}

// 故障等级(紧急程度)。
export const FAULT_LEVEL = {
  low: { label: '一般', type: 'info' },
  normal: { label: '普通', type: 'primary' },
  high: { label: '紧急', type: 'warning' },
  urgent: { label: '特急', type: 'danger' },
}

// 故障来源。
export const FAULT_SOURCE = {
  inspection: { label: '巡检发现', type: 'primary' },
  citizen: { label: '市民上报', type: 'warning' },
  monitoring: { label: '系统告警', type: 'danger' },
  other: { label: '其它', type: 'info' },
}

// 维修记录状态。
export const REPAIR_STATUS = {
  ongoing: { label: '维修中', type: 'warning' },
  finished: { label: '已完成', type: 'success' },
}

// 维修结果。
export const REPAIR_RESULT = {
  fixed: { label: '已修复', type: 'success' },
  pending_parts: { label: '待配件', type: 'warning' },
  observing: { label: '观察中', type: 'primary' },
  unfixable: { label: '无法修复', type: 'danger' },
}

// 追踪时间线的节点名称。
export const TIMELINE_STAGE = {
  reported: { label: '故障登记', type: 'primary' },
  repair_started: { label: '维修开工', type: 'warning' },
  repair_finished: { label: '维修完成', type: 'success' },
  closed: { label: '故障关闭', type: 'info' },
}

// 取字典项文案。
export function dictLabel(dict, key, fallback = '-') {
  if (key === null || key === undefined || key === '') return fallback
  return dict[key]?.label ?? key
}

// 取字典项标签类型。
export function dictType(dict, key, fallback = 'info') {
  return dict[key]?.type ?? fallback
}

// 将字典转换为下拉选项。
export function dictOptions(dict) {
  return Object.entries(dict).map(([value, item]) => ({ value, label: item.label }))
}
