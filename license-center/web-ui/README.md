# License Center Web UI

现代化的授权管理系统前端界面

## 技术栈

- **框架**: React 18 + TypeScript
- **构建工具**: Vite
- **UI 组件**: Ant Design 5
- **状态管理**: Zustand
- **数据请求**: TanStack Query (React Query)
- **HTTP 客户端**: Axios
- **路由**: React Router v6
- **图表**: Recharts
- **日期处理**: Day.js

## 功能特性

### 🎨 页面
- **登录页面**: 用户认证
- **仪表盘**: 实时统计数据和趋势图表
- **授权码管理**: 完整的 CRUD 操作、批量操作、转移功能
- **套餐管理**: 套餐配置和版本限制
- **告警管理**: 实时告警查看和处理
- **Webhook 管理**: 事件通知配置
- **标签管理**: 授权码分类标签
- **验证日志**: 完整的验证记录查询

### ✨ 特性
- 响应式设计，支持移动端
- 暗色侧边栏导航
- 实时数据刷新
- 分页和筛选
- 批量操作
- 表单验证
- 错误处理
- Token 持久化

## 开发

```bash
# 安装依赖
npm install

# 启动开发服务器
npm run dev

# 构建生产版本
npm run build

# 预览生产构建
npm run preview
```

## 项目结构

```
src/
├── api/              # API 接口定义
├── components/       # 可复用组件
├── layouts/          # 布局组件
├── pages/            # 页面组件
│   ├── Login.tsx
│   ├── Dashboard.tsx
│   ├── Licenses.tsx
│   ├── Plans.tsx
│   ├── Alerts.tsx
│   ├── Webhooks.tsx
│   ├── Tags.tsx
│   └── Logs.tsx
├── store/            # 状态管理
├── types/            # TypeScript 类型定义
├── utils/            # 工具函数
├── App.tsx           # 应用入口
└── main.tsx          # 主入口
```

## 配置

### API 代理

开发环境下，API 请求会自动代理到 `http://localhost:8090`

```typescript
// vite.config.ts
server: {
  proxy: {
    '/api': {
      target: 'http://localhost:8090',
      changeOrigin: true,
    },
  },
}
```

### 环境变量

创建 `.env.local` 文件：

```env
VITE_API_BASE_URL=http://localhost:8090
```

## 部署

### 构建

```bash
npm run build
```

构建产物在 `dist/` 目录

### Nginx 配置示例

```nginx
server {
    listen 80;
    server_name license.yourdomain.com;

    root /path/to/dist;
    index index.html;

    location / {
        try_files $uri $uri/ /index.html;
    }

    location /api {
        proxy_pass http://localhost:8090;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

## 集成到 Go 后端

可以将构建后的静态文件嵌入到 Go 程序中：

```go
//go:embed dist
var webUI embed.FS

func setupWebUI(r *gin.Engine) {
    r.StaticFS("/console", http.FS(webUI))
}
```

## 浏览器支持

- Chrome >= 90
- Firefox >= 88
- Safari >= 14
- Edge >= 90

## 开发规范

- 使用 TypeScript 严格模式
- 遵循 React Hooks 最佳实践
- 使用 TanStack Query 管理服务端状态
- 使用 Zustand 管理客户端状态
- 组件按功能模块组织
- API 调用统一在 `api/` 目录管理

## 性能优化

- 代码分割（按路由）
- 懒加载组件
- 图片优化
- 缓存策略
- Tree Shaking
- 生产环境压缩

## 安全

- JWT Token 认证
- 请求拦截器自动添加 Token
- 401 自动跳转登录
- XSS 防护
- CSRF 防护

## License

MIT
