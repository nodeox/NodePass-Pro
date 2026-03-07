import type { UserRole } from '../types'

export type PortalType = 'user' | 'admin'

const PORTAL_PREFIX_MAP: Record<PortalType, string> = {
  user: '/user',
  admin: '/admin',
}

export const resolvePortalByPathname = (pathname: string): PortalType =>
  pathname.startsWith(PORTAL_PREFIX_MAP.admin) ? 'admin' : 'user'

export const getPortalPrefix = (portal: PortalType): string =>
  PORTAL_PREFIX_MAP[portal]

export const buildPortalPath = (portal: PortalType, path: string): string => {
  const normalized = path.startsWith('/') ? path : `/${path}`
  return `${getPortalPrefix(portal)}${normalized}`
}

export const getHomePathByRole = (role?: UserRole | null): string =>
  role === 'admin' ? '/admin/dashboard' : '/user/dashboard'

