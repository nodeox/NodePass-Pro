/**
 * Vite 插件：添加安全响应头
 *
 * 包含：
 * - Content-Security-Policy (CSP)
 * - X-Content-Type-Options
 * - X-Frame-Options
 * - X-XSS-Protection
 * - Referrer-Policy
 */

import type { Plugin } from 'vite'

export function securityHeadersPlugin(): Plugin {
  return {
    name: 'security-headers',
    configureServer(server) {
      server.middlewares.use((_req, res, next) => {
        // Content Security Policy
        // 注意：开发环境需要允许 'unsafe-eval' 用于 HMR
        const isDev = process.env.NODE_ENV !== 'production'
        const cspDirectives = [
          "default-src 'self'",
          isDev
            ? "script-src 'self' 'unsafe-eval' 'unsafe-inline'"
            : "script-src 'self'",
          "style-src 'self' 'unsafe-inline'", // Ant Design 需要 inline styles
          "img-src 'self' data: https:",
          "font-src 'self' data:",
          "connect-src 'self' ws: wss:", // WebSocket 连接
          "frame-ancestors 'none'",
          "base-uri 'self'",
          "form-action 'self'",
        ]

        res.setHeader('Content-Security-Policy', cspDirectives.join('; '))

        // 防止 MIME 类型嗅探
        res.setHeader('X-Content-Type-Options', 'nosniff')

        // 防止点击劫持
        res.setHeader('X-Frame-Options', 'DENY')

        // XSS 保护（虽然现代浏览器已内置，但仍建议设置）
        res.setHeader('X-XSS-Protection', '1; mode=block')

        // Referrer 策略
        res.setHeader('Referrer-Policy', 'strict-origin-when-cross-origin')

        // 禁用浏览器功能
        res.setHeader(
          'Permissions-Policy',
          'geolocation=(), microphone=(), camera=()',
        )

        next()
      })
    },
  }
}
