import dayjs from 'dayjs'

export const formatTraffic = (bytes: number): string => {
  if (!Number.isFinite(bytes) || bytes <= 0) {
    return '0 B'
  }

  const units = ['B', 'KB', 'MB', 'GB', 'TB', 'PB']
  let index = 0
  let current = bytes

  while (current >= 1024 && index < units.length - 1) {
    current /= 1024
    index += 1
  }

  if (index === 0) {
    return `${Math.round(current)} ${units[index]}`
  }

  return `${current.toFixed(current >= 100 ? 0 : 1)} ${units[index]}`
}

export const formatBytes = (value: number): string => {
  return formatTraffic(value)
}

export const formatDateTime = (value?: string | null): string => {
  if (!value) {
    return '-'
  }
  return dayjs(value).format('YYYY-MM-DD HH:mm:ss')
}
