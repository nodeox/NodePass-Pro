# NodePass License Center 升级指南

## 🚀 一键升级脚本

### 快速升级

```bash
# 升级到最新版本
./upgrade.sh

# 升级到指定版本
./upgrade.sh 1.0.0
```

## 📋 升级流程

升级脚本会自动执行以下步骤：

1. ✅ **检查 Docker 环境**
2. ✅ **检查当前版本**
3. ✅ **备份数据和配置**
4. ✅ **停止旧容器**
5. ✅ **删除旧容器**
6. ✅ **拉取新镜像**
7. ✅ **启动新容器**
8. ✅ **健康检查**
9. ✅ **清理旧镜像**（可选）

## 🔒 安全特性

### 自动备份

升级前会自动备份：
- 数据库文件 (`./data`)
- 配置文件 (`./configs`)
- 容器配置

备份位置：`./backups/backup_YYYYMMDD_HHMMSS/`

### 自动回滚

如果升级失败，脚本会自动：
1. 停止新容器
2. 恢复备份数据
3. 启动旧版本容器

### 健康检查

升级后会自动检查服务健康状态，确保服务正常运行。

## 📖 使用说明

### 基本用法

```bash
# 1. 进入项目目录
cd /path/to/license-center

# 2. 设置环境变量（如果需要）
export JWT_SECRET="your-jwt-secret"
export ADMIN_PASSWORD="your-admin-password"

# 3. 执行升级
./upgrade.sh latest
```

### 升级到指定版本

```bash
# 升级到 1.0.0
./upgrade.sh 1.0.0

# 升级到 1.1.0
./upgrade.sh 1.1.0

# 升级到最新版本
./upgrade.sh latest
```

### 环境变量

升级脚本支持以下环境变量：

```bash
# JWT 密钥（必需）
export JWT_SECRET="your-secret-key"

# 管理员密码（必需）
export ADMIN_PASSWORD="your-password"

# 容器名称（可选，默认：license-center）
export CONTAINER_NAME="license-center"

# 镜像名称（可选，默认：ghcr.io/nodeox/license-center）
export IMAGE_NAME="ghcr.io/nodeox/license-center"
```

## 🔄 升级示例

### 示例 1: 从 1.0.0 升级到 1.1.0

```bash
$ ./upgrade.sh 1.1.0

╔════════════════════════════════════════════════╗
║                                                ║
║   NodePass License Center 一键升级脚本         ║
║                                                ║
╚════════════════════════════════════════════════╝

[INFO] 目标版本: 1.1.0

是否继续升级? (y/N) y

[STEP] 检查 Docker 环境...
[INFO] Docker 环境检查通过 ✓

[STEP] 检查当前版本...
[INFO] 当前版本: ghcr.io/nodeox/license-center:1.0.0
[INFO] 容器状态: running

[STEP] 备份数据...
[INFO] 备份数据库...
[INFO] 数据库备份完成: ./backups/backup_20260309_120000/data
[INFO] 备份配置文件...
[INFO] 配置文件备份完成: ./backups/backup_20260309_120000/configs
[SUCCESS] 备份完成: ./backups/backup_20260309_120000

[STEP] 停止旧容器...
[INFO] 停止容器: license-center
[INFO] 容器已停止 ✓

[STEP] 删除旧容器...
[INFO] 删除容器: license-center
[INFO] 容器已删除 ✓

[STEP] 拉取新镜像...
[INFO] 镜像: ghcr.io/nodeox/license-center:1.1.0
1.1.0: Pulling from nodeox/license-center
...
[SUCCESS] 镜像拉取成功 ✓

[STEP] 启动新容器...
[INFO] 启动容器: license-center
[SUCCESS] 容器已启动 ✓

[STEP] 执行健康检查...
[INFO] 等待服务启动...
..........
[SUCCESS] 健康检查通过 ✓

[STEP] 清理旧镜像...
是否清理旧的 Docker 镜像? (y/N) y
[INFO] 清理未使用的镜像...
[SUCCESS] 清理完成 ✓

╔════════════════════════════════════════════════╗
║          升级完成！                             ║
╚════════════════════════════════════════════════╝

[INFO] 升级信息:
  旧版本: ghcr.io/nodeox/license-center:1.0.0
  新版本: ghcr.io/nodeox/license-center:1.1.0
  备份位置: ./backups/backup_20260309_120000

[INFO] 访问地址:
  http://localhost:8090
  http://localhost:8090/console

[SUCCESS] 升级流程完成！
```

### 示例 2: 升级失败自动回滚

```bash
$ ./upgrade.sh 2.0.0

...
[STEP] 执行健康检查...
[INFO] 等待服务启动...
..............................
[ERROR] 健康检查失败
[WARN] 查看容器日志:
...
[WARN] 尝试从备份恢复...
[ERROR] 升级失败，开始回滚...
[INFO] 从备份恢复: ./backups/backup_20260309_120000
[INFO] 数据已恢复
[INFO] 配置已恢复
[INFO] 启动旧版本容器: ghcr.io/nodeox/license-center:1.0.0
[SUCCESS] 已回滚到旧版本
```

## 🛠️ 手动升级

如果需要手动升级，可以按照以下步骤：

### 1. 备份数据

```bash
# 创建备份目录
mkdir -p backups/manual_backup_$(date +%Y%m%d_%H%M%S)

# 备份数据
cp -r data backups/manual_backup_$(date +%Y%m%d_%H%M%S)/
cp -r configs backups/manual_backup_$(date +%Y%m%d_%H%M%S)/
```

### 2. 停止并删除旧容器

```bash
docker stop license-center
docker rm license-center
```

### 3. 拉取新镜像

```bash
docker pull ghcr.io/nodeox/license-center:1.1.0
```

### 4. 启动新容器

```bash
docker run -d \
  --name license-center \
  -p 8090:8090 \
  -v $(pwd)/data:/app/data \
  -v $(pwd)/configs:/app/configs \
  -e JWT_SECRET="your-secret" \
  -e ADMIN_PASSWORD="your-password" \
  --restart unless-stopped \
  ghcr.io/nodeox/license-center:1.1.0
```

### 5. 验证升级

```bash
# 检查容器状态
docker ps | grep license-center

# 健康检查
curl http://localhost:8090/health

# 查看版本
curl http://localhost:8090/
```

## 🔙 回滚到旧版本

### 使用备份回滚

```bash
# 1. 停止当前容器
docker stop license-center
docker rm license-center

# 2. 恢复备份
BACKUP_DIR="./backups/backup_20260309_120000"
rm -rf data configs
cp -r ${BACKUP_DIR}/data ./
cp -r ${BACKUP_DIR}/configs ./

# 3. 启动旧版本
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

## 📊 升级前检查清单

在升级前，请确认：

- [ ] 已备份重要数据
- [ ] 已阅读新版本的更新日志
- [ ] 已测试新版本的兼容性
- [ ] 已设置正确的环境变量
- [ ] 有足够的磁盘空间
- [ ] 在非高峰时段进行升级

## ⚠️ 注意事项

### 1. 数据库迁移

某些版本升级可能需要数据库迁移。升级前请查看版本更新日志。

### 2. 配置文件变更

新版本可能引入新的配置项。升级后请检查配置文件。

### 3. 端口冲突

确保 8090 端口未被其他服务占用。

### 4. 权限问题

确保有足够的权限访问数据目录和配置目录。

### 5. 网络问题

确保能够访问 GitHub Container Registry (ghcr.io)。

## 🔍 故障排查

### 问题 1: 镜像拉取失败

**原因**: 网络问题或权限不足

**解决**:
```bash
# 检查网络
ping ghcr.io

# 登录 GitHub Container Registry
export CR_PAT=YOUR_TOKEN
echo $CR_PAT | docker login ghcr.io -u YOUR_USERNAME --password-stdin

# 重试升级
./upgrade.sh 1.1.0
```

### 问题 2: 健康检查失败

**原因**: 服务启动失败或配置错误

**解决**:
```bash
# 查看容器日志
docker logs license-center

# 检查配置文件
cat configs/config.yaml

# 检查环境变量
docker inspect license-center | grep -A 10 Env
```

### 问题 3: 数据丢失

**原因**: 备份失败或恢复错误

**解决**:
```bash
# 查看备份列表
ls -lh backups/

# 手动恢复备份
BACKUP_DIR="./backups/backup_YYYYMMDD_HHMMSS"
cp -r ${BACKUP_DIR}/data ./
cp -r ${BACKUP_DIR}/configs ./
```

### 问题 4: 端口被占用

**原因**: 8090 端口已被使用

**解决**:
```bash
# 查看端口占用
lsof -i :8090

# 停止占用端口的进程
kill -9 PID

# 或使用其他端口
docker run -d -p 8091:8090 ...
```

## 📚 相关文档

- `DOCKER_BUILD_REPORT.md` - Docker 镜像构建报告
- `GITHUB_REGISTRY_GUIDE.md` - GitHub Container Registry 指南
- `CHANGELOG.md` - 版本更新日志

## 🆘 获取帮助

如果遇到问题：

1. 查看容器日志：`docker logs license-center`
2. 查看备份目录：`ls -lh backups/`
3. 查看升级脚本帮助：`./upgrade.sh --help`
4. 提交 Issue：https://github.com/nodeox/nodepass/issues

---

**升级前请务必备份数据！**
