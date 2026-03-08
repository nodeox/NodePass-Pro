# 个人中心增强 - 快速开始

## 安装和运行

### 前端开发

```bash
cd /Users/jianshe/Projects/NodePass-Pro/frontend

# 安装依赖（如果需要）
npm install

# 启动开发服务器
npm run dev

# 访问 http://localhost:5173
```

### 构建生产版本

```bash
cd /Users/jianshe/Projects/NodePass-Pro/frontend

# 构建
npm run build

# 构建产物在 dist/ 目录
```

## 访问个人中心

1. 启动前端和后端服务
2. 登录系统
3. 点击顶部导航栏的"个人中心"或访问 `/profile`

## 功能测试清单

### ✅ 基本信息展示
- [ ] 查看用户名、邮箱、角色
- [ ] 查看 VIP 等级和到期时间
- [ ] 查看流量配额和使用情况
- [ ] 流量进度条显示正确

### ✅ 修改密码
- [ ] 点击"修改密码"按钮
- [ ] 输入原密码（错误密码应提示错误）
- [ ] 输入新密码（少于8个字符应提示错误）
- [ ] 确认新密码（不一致应提示错误）
- [ ] 成功修改密码
- [ ] 使用新密码可以登录

### ✅ 流量统计
- [ ] 切换到"流量统计"标签
- [ ] 查看统计卡片（入站、出站、计费）
- [ ] 查看趋势图
- [ ] 选择不同时间范围
- [ ] 图表数据更新

### ✅ 安全设置
- [ ] 查看 Telegram 绑定状态
- [ ] 点击"撤销所有登录会话"
- [ ] 确认操作
- [ ] 自动跳转到登录页
- [ ] 需要重新登录

### 🚧 修改邮箱（开发中）
- [ ] 点击邮箱旁的"修改"按钮
- [ ] 输入新邮箱
- [ ] 输入密码确认
- [ ] 显示"功能开发中"提示

## 常见问题

### Q: 修改密码后是否需要重新登录？
A: 不需要。修改密码后当前会话仍然有效，但建议使用"撤销所有登录会话"功能确保其他设备登出。

### Q: 流量统计数据多久更新一次？
A: 流量数据按小时聚合，可能存在一定延迟。

### Q: 撤销所有会话后会发生什么？
A: 所有设备上的登录会话都会失效，包括当前设备。您需要在所有设备上重新登录。

### Q: 修改邮箱功能什么时候可用？
A: 前端界面已完成，后端 API 正在开发中。

### Q: 如何启用两步验证？
A: 两步验证功能正在开发中，敬请期待。

## 开发说明

### 添加新的统计维度

编辑 `frontend/src/pages/profile/components/TrafficChart.tsx`：

```typescript
// 添加新的统计卡片
<Col span={6}>
  <Card>
    <Statistic
      title="新维度"
      value={newValue}
      formatter={(value) => formatValue(Number(value))}
      prefix={<NewIcon />}
    />
  </Card>
</Col>
```

### 添加新的安全选项

编辑 `frontend/src/pages/profile/components/SecuritySettings.tsx`：

```typescript
<div>
  <h4>新安全选项</h4>
  <p>描述</p>
  <Button onClick={handleNewAction}>
    执行操作
  </Button>
</div>
```

### 修改图表样式

编辑 `TrafficChart.tsx` 中的 `option` 对象：

```typescript
const option = {
  // 修改颜色
  series: [
    {
      itemStyle: { color: '#your-color' },
    },
  ],
  // 修改其他配置
}
```

## API 端点

### 已使用的 API

- `GET /api/v1/auth/me` - 获取当前用户信息
- `PUT /api/v1/auth/password` - 修改密码
- `POST /api/v1/auth/revoke-all` - 撤销所有会话
- `GET /api/v1/traffic/usage` - 获取流量汇总
- `GET /api/v1/traffic/records` - 获取流量记录

### 待实现的 API

- `PUT /api/v1/auth/email` - 修改邮箱（建议）
- `POST /api/v1/auth/2fa/enable` - 启用两步验证（建议）
- `GET /api/v1/auth/sessions` - 获取登录会话列表（建议）

## 性能优化建议

1. **图表数据缓存**
   ```typescript
   // 使用 useMemo 缓存图表配置
   const option = useMemo(() => ({
     // 配置
   }), [trafficData])
   ```

2. **懒加载标签页**
   ```typescript
   // 只在标签页激活时加载数据
   useEffect(() => {
     if (activeTab === 'traffic') {
       fetchTrafficData()
     }
   }, [activeTab])
   ```

3. **防抖搜索**
   ```typescript
   // 使用 debounce 优化时间范围选择
   const debouncedFetch = useMemo(
     () => debounce(fetchTrafficData, 500),
     []
   )
   ```

## 故障排查

### 图表不显示
- 检查是否有流量数据
- 检查浏览器控制台是否有错误
- 确认 ECharts 库已正确加载

### API 调用失败
- 检查后端服务是否运行
- 检查 JWT token 是否有效
- 查看网络请求的响应状态码

### 样式问题
- 清除浏览器缓存
- 检查 Ant Design 版本兼容性
- 确认 CSS 文件已正确导入

## 相关文档

- [功能详细说明](./profile-enhancement.md)
- [变更摘要](./CHANGELOG-profile-enhancement.md)
- [UI 预览](./profile-ui-preview.md)
- [后端 API 文档](../backend/README.md)

## 联系方式

如有问题或建议，请提交 Issue 或 Pull Request。
