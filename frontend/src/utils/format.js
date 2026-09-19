function pad(value) {
  return String(value).padStart(2, '0')
}

function toDate(value) {
  if (!value) return null
  const date = value instanceof Date ? value : new Date(value)
  return Number.isNaN(date.getTime()) ? null : date
}

// 格式化为 YYYY-MM-DD HH:mm。
export function formatDateTime(value) {
  const date = toDate(value)
  if (!date) return '-'
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}`
}

// 格式化为 YYYY-MM-DD。
export function formatDate(value) {
  const date = toDate(value)
  if (!date) return '-'
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}`
}

// 维修耗时(分钟)转可读文案。
export function formatDuration(minutes) {
  if (minutes === null || minutes === undefined || minutes === '') return '-'
  const value = Number(minutes)
  if (Number.isNaN(value)) return '-'
  if (value < 60) return `${value} 分钟`
  const hours = Math.floor(value / 60)
  const rest = value % 60
  return rest > 0 ? `${hours} 小时 ${rest} 分钟` : `${hours} 小时`
}

// 小时数保留一位小数。
export function formatHours(value) {
  const num = Number(value ?? 0)
  return Number.isNaN(num) ? '-' : `${num.toFixed(1)} 小时`
}

// 金额格式化。
export function formatMoney(value) {
  const num = Number(value ?? 0)
  return Number.isNaN(num) ? '-' : `¥ ${num.toFixed(2)}`
}

// 计算已等待时长文案。
export function formatWaiting(hours) {
  const num = Number(hours ?? 0)
  if (Number.isNaN(num)) return '-'
  if (num < 1) return '不足 1 小时'
  if (num < 24) return `${num.toFixed(1)} 小时`
  return `${(num / 24).toFixed(1)} 天`
}
