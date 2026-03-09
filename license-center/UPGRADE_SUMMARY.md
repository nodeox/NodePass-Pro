# 一键升级脚本 - 完成报告

## ✅ 已创建的升级脚本

### 1. upgrade.sh - 完整升级脚本（推荐）

**功能特性**：
- ✅ 自动备份数据和配置
- ✅ 健康检查
- ✅ 失败自动回滚
- ✅ 交互式确认
- ✅ 详细的日志输出
- ✅ 清理旧镜像（可选）

**使用方法**：
```bash
# 升级到最新版本
./upgrade.sh

# 升级到指定版本
./upgrade.sh 1.0.0
```

### 2. quick-upgrade.sh - 快速升级脚本

**功能特性**：
- ✅ 快速升级（无交互）
- ✅ 自动备份
- ✅ 健康检查
- ⚠️ 无自动回滚

**使用方法**：
```bash
# 快速升级到最新版本
./quick-upgrade.sh

# 快速升级到指定版本
./quick-upgrade.sh 1.0.0
```

## 📋 升级流程对比

| 功能 | upgrade.sh | quick-upgrade.sh |
|------|-----------|------------------|
| 自动备份 | ✅ | ✅ |
| 健康检查 | ✅ | ✅ |
| 自动回滚 | ✅ | ❌ |
| 交互确认 | ✅ | ❌ |
| 详细日志 | ✅ | 简化 |
| 清理旧镜像 | ✅ | ❌ |
| 执行速度 | 较慢 | 快速 |
| 推荐场景 | 生产环境 | 开发/测试 |

## 🚀 快速开始

### 生产环境升级（推荐）

```bash
cd /Users/jianshe/Projects/NodePass-Pro/license-center

# 设置环境变量
export JWT_SECRET="your-jwt-secret"
export ADMIN_PASSWORD="your-admin-password"

# 执行升级
./upgrade.sh latest
```

### 开发环境快速升级

```bash
cd /Users/jianshe/Projects/NodePass-Pro/license-center

# 快速升级
./quick-upgrade.sh latest
```

## 📖 详细使用说明

### upgrade.sh 完整流程

```bash
$ ./upgrade.sh 1.0.0

╔════════════════════════════════════════════════╗
║   NodePass License Center 一键升级脚本         ║
╚════════════════════════════════════════════════╝

[INFO] 目标版本: 1.0.0

是否继续升级? (y/N) y

[STEP] 检查 Docker 环境...
[INFO] Docker 环境检查通过 ✓

[STEP] 检查当前版本...
[INFO] 当前版本: ghcr.io/nodeox/license-center:latest
[INFO] 容器状态: running

[STEP] 备份数据...
[INFO] 备份数据库...
[INFO] 备份配置文件...
[SUCCESS] 备份完成: ./backups/backup_20260309_120000

[STEP] 停止旧容器...
[INFO] 容器已停止 ✓

[STEP] 删除旧容器...
[INFO] 容器已删除 ✓

[STEP] 拉取新镜像...
[SUCCESS] 镜像拉取成功 ✓

[STEP] 启动新容器...
[SUCCESS] 容器已启动 ✓

[STEP] 执行健康检查...
[SUCCESS] 健康检查通过 ✓

[STEP] 清理旧镜像...
是否清理旧的 Docker 镜像? (y/N) y
[SUCCESS] 清理完成 ✓

╔════════════════════════════════════════════════╗
║          升级完成！                             ║
╚════════════════════════════════════════════════╝

[INFO] 升级信息:
  旧版本: ghcr.io/nodeox/license-center:latest
  新版本: ghcr.io/nodeox/license-center:1.0.0
  备份位置: ./backups/backup_20260309_120000

[INFO] 访问地址:
  http://localhost:8090
  http://localhost:8090/console

[SUCCESS] 升级流程完成！
```

## 🔒 安全特性

### 1. 自动备份

升级前自动备份：
- 数据库文件 (`./data`)
- 配置文件 (`./configs`)
- 容器配置信息

备份位置：`./backups/backup_YYYYMMDD_HHMMSS/`

### 2. 健康检查

升级后自动检查：
- 服务是否启动
- 健康检查端点是否响应
- 版本信息是否正确

### 3. 自动回滚

如果升级失败，`upgrade.sh` 会自动：
1. 停止新容器
2. 恢复备份数据
3. 启动旧版本容器

## 📊 升级前检查清单

- [ ] 已备份重要数据
- [ ] 已设置环境变量（JWT_SECRET, ADMIN_PASSWORD）
- [ ] 已阅读版本更新日志
- [ ] 有足够的磁盘空间
- [ ] 在非高峰时段升级
- [ ] 已通知相关人员

## 🔄 升级场景

### 场景 1: 首次部署

```bash
# 拉取镜像
docker pull ghcr.io/nodeox/license-center:latest

# 运行容器
docker run -d \
  --name license-center \
  -p 8090:8090 \
  -v $(pwd)/data:/app/data \
  -v $(pwd)/configs:/app/configs \
  -e JWT_SECRET="your-secret" \
  -e ADMIN_PASSWORD="your-password" \
  --restart unless-stopped \
  ghcr.io/nodeox/license-center:latest
```

### 场景 2: 版本升级

```bash
# 使用升级脚本
./upgrade.sh 1.1.0
```

### 场景 3: 紧急回滚

```bash
# 查看备份
ls -lh backups/

# 使用最新备份回滚
BACKUP_DIR="./backups/backup_20260309_120000"

# 停止当前容器
docker stop license-center
docker rm license-center

# 恢复数据
rm -rf data configs
cp -r ${BACKUP_DIR}/data ./
cp -r ${BACKUP_DIR}/configs ./

# 启动旧版本
docker run -d \
  --name license-center \
  -p 8090:8090 \
  -v $(pwd)/data:/app/data \
  -v $(pwd)/configs:/app/configs \
  -e JWT_SECRET="your-secret" \
  -e ADMIN_PASSWORD="your-password" \
  --restart unless-stopped \
  ghcr.io/nodeox/license-center:1.0.0
```

## 🛠️ 环境变量

### 必需的环境变量

```bash
# JWT 密钥（强烈建议设置）
export JWT_SECRET="your-strong-random-secret-key"

# 管理员密码（强烈建议设置）
export ADMIN_PASSWORD="your-strong-password"
```

### 可选的环境变量

```bash
# 容器名称（默认：license-center）
export CONTAINER_NAME="license-center"

# 镜像名称（默认：ghcr.io/nodeox/license-center）
export IMAGE_NAME="ghcr.io/nodeox/license-center"

# 数据目录（默认：./data）
export DATA_DIR="./data"

# 配置目录（默认：./configs）
export CONFIG_DIR="./configs"

# 备份目录（默认：./backups）
export BACKUP_DIR="./backups"
```

## 📝 升级日志

升级过程中的所有操作都会记录在终端输出中。建议保存升级日志：

```bash
# 保存升级日志
./upgrade.sh 1.1.0 2>&1 | tee upgrade_$(date +%Y%m%d_%H%M%S).log
```

## ⚠️ 注意事项

1. **生产环境升级**：建议在非高峰时段进行
2. **数据备份**：升级前会自动备份，但建议额外手动备份
3. **版本兼容性**：查看版本更新日志了解兼容性变更
4. **环境变量**：确保设置了正确的 JWT_SECRET 和 ADMIN_PASSWORD
5. **磁盘空间**：确保有足够的空间存储备份和新镜像

## 🔍 故障排查

### 查看容器日志

```bash
docker logs license-center
docker logs -f license-center  # 实时查看
docker logs --tail 100 license-center  # 查看最后100行
```

### 查看容器状态

```bash
docker ps | grep license-center
docker inspect license-center
```

### 查看备份

```bash
ls -lh backups/
du -sh backups/*
```

### 测试健康检查

```bash
curl http://localhost:8090/health
curl http://localhost:8090/
```

## 📚 相关文档

- `UPGRADE_GUIDE.md` - 详细升级指南
- `DOCKER_BUILD_REPORT.md` - Docker 镜像构建报告
- `GITHUB_REGISTRY_GUIDE.md` - GitHub Container Registry 指南
- `upgrade.sh` - 完整升级脚本
- `quick-upgrade.sh` - 快速升级脚本

## 🎯 最佳实践

1. **定期升级**：保持系统更新到最新稳定版本
2. **测试环境**：先在测试环境验证升级
3. **备份策略**：定期备份数据，保留多个备份版本
4. **监控告警**：升级后监控系统运行状态
5. **文档记录**：记录每次升级的时间、版本和问题

## 🆘 获取帮助

如果遇到问题：

1. 查看升级指南：`UPGRADE_GUIDE.md`
2. 查看容器日志：`docker logs license-center`
3. 查看备份目录：`ls -lh backups/`
4. 提交 Issue：https://github.com/nodeox/nodepass/issues

---

**升级脚本已准备就绪，可以开始使用！** 🚀
