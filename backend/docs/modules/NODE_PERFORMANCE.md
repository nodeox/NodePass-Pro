# 节点性能监控模块文档

## 概述

节点性能监控模块负责收集节点的性能指标（CPU、内存、磁盘、网络），提供告警配置和统计分析功能。

## 功能特性

- ✅ 性能指标收集（CPU、内存、磁盘、网络）
- ✅ 性能告警配置
- ✅ 阈值检查与告警触发
- ✅ 统计分析（平均值、最大值、最小值）
- ✅ 时间范围查询
- ✅ 趋势分析

## 架构设计

### 领域层

```go
// PerformanceMetric 性能指标实体
type PerformanceMetric struct {
    ID             uint
    NodeInstanceID uint      // 节点实例 ID
    CPUUsage       float64   // CPU 使用率 (0-100)
    MemoryUsage    float64   // 内存使用率 (0-100)
    DiskUsage      float64   // 磁盘使用率 (0-100)
    NetworkIn      uint64    // 入站流量 (bytes)
    NetworkOut     uint64    // 出站流量 (bytes)
    RecordedAt     time.Time // 记录时间
}

// PerformanceAlert 性能告警配置
type PerformanceAlert struct {
    ID              uint
    NodeInstanceID  uint
    Enabled         bool
    CPUThreshold    float64  // CPU 告警阈值
    MemoryThreshold float64  // 内存告警阈值
    DiskThreshold   float64  // 磁盘告警阈值
    NetworkInLimit  uint64   // 入站流量限制
    NetworkOutLimit uint64   // 出站流量限制
    CreatedAt       time.Time
    UpdatedAt       time.Time
}

// PerformanceStats 性能统计
type PerformanceStats struct {
    NodeInstanceID uint
    StartTime      time.Time
    EndTime        time.Time
    AvgCPU         float64
    MaxCPU         float64
    MinCPU         float64
    AvgMemory      float64
    MaxMemory      float64
    MinMemory      float64
    AvgDisk        float64
    MaxDisk        float64
    MinDisk        float64
    TotalNetworkIn uint64
    TotalNetworkOut uint64
}
```

### 业务规则

#### 1. 指标验证

```go
func (m *PerformanceMetric) Validate() error {
    if m.CPUUsage < 0 || m.CPUUsage > 100 {
        return ErrInvalidCPUUsage
    }
    if m.MemoryUsage < 0 || m.MemoryUsage > 100 {
        return ErrInvalidMemoryUsage
    }
    if m.DiskUsage < 0 || m.DiskUsage > 100 {
        return ErrInvalidDiskUsage
    }
    return nil
}
```

#### 2. 阈值检查

```go
func (a *PerformanceAlert) CheckThresholds(metric *PerformanceMetric) []string {
    var alerts []string

    if !a.Enabled {
        return alerts
    }

    if metric.CPUUsage > a.CPUThreshold {
        alerts = append(alerts, "cpu")
    }
    if metric.MemoryUsage > a.MemoryThreshold {
        alerts = append(alerts, "memory")
    }
    if metric.DiskUsage > a.DiskThreshold {
        alerts = append(alerts, "disk")
    }
    if metric.NetworkIn > a.NetworkInLimit {
        alerts = append(alerts, "network_in")
    }
    if metric.NetworkOut > a.NetworkOutLimit {
        alerts = append(alerts, "network_out")
    }

    return alerts
}
```

#### 3. 统计计算

```go
func CalculateStats(metrics []*PerformanceMetric) *PerformanceStats {
    if len(metrics) == 0 {
        return nil
    }

    stats := &PerformanceStats{
        NodeInstanceID: metrics[0].NodeInstanceID,
        StartTime:      metrics[0].RecordedAt,
        EndTime:        metrics[len(metrics)-1].RecordedAt,
    }

    var sumCPU, sumMemory, sumDisk float64
    stats.MaxCPU = metrics[0].CPUUsage
    stats.MinCPU = metrics[0].CPUUsage
    stats.MaxMemory = metrics[0].MemoryUsage
    stats.MinMemory = metrics[0].MemoryUsage
    stats.MaxDisk = metrics[0].DiskUsage
    stats.MinDisk = metrics[0].DiskUsage

    for _, m := range metrics {
        // CPU
        sumCPU += m.CPUUsage
        if m.CPUUsage > stats.MaxCPU {
            stats.MaxCPU = m.CPUUsage
        }
        if m.CPUUsage < stats.MinCPU {
            stats.MinCPU = m.CPUUsage
        }

        // Memory
        sumMemory += m.MemoryUsage
        if m.MemoryUsage > stats.MaxMemory {
            stats.MaxMemory = m.MemoryUsage
        }
        if m.MemoryUsage < stats.MinMemory {
            stats.MinMemory = m.MemoryUsage
        }

        // Disk
        sumDisk += m.DiskUsage
        if m.DiskUsage > stats.MaxDisk {
            stats.MaxDisk = m.DiskUsage
        }
        if m.DiskUsage < stats.MinDisk {
            stats.MinDisk = m.DiskUsage
        }

        // Network
        stats.TotalNetworkIn += m.NetworkIn
        stats.TotalNetworkOut += m.NetworkOut
    }

    count := float64(len(metrics))
    stats.AvgCPU = sumCPU / count
    stats.AvgMemory = sumMemory / count
    stats.AvgDisk = sumDisk / count

    return stats
}
```

### 应用层

#### Commands (命令)

**RecordMetricCommand** - 记录性能指标
```go
type RecordMetricCommand struct {
    NodeInstanceID uint
    CPUUsage       float64
    MemoryUsage    float64
    DiskUsage      float64
    NetworkIn      uint64
    NetworkOut     uint64
}
```

**CreateAlertCommand** - 创建告警配置
```go
type CreateAlertCommand struct {
    NodeInstanceID  uint
    Enabled         bool
    CPUThreshold    float64
    MemoryThreshold float64
    DiskThreshold   float64
    NetworkInLimit  uint64
    NetworkOutLimit uint64
}
```

**UpdateAlertCommand** - 更新告警配置
```go
type UpdateAlertCommand struct {
    NodeInstanceID  uint
    Enabled         *bool
    CPUThreshold    *float64
    MemoryThreshold *float64
    DiskThreshold   *float64
    NetworkInLimit  *uint64
    NetworkOutLimit *uint64
}
```

#### Queries (查询)

**GetMetricsQuery** - 获取性能指标
```go
type GetMetricsQuery struct {
    NodeInstanceID uint
    StartTime      time.Time
    EndTime        time.Time
    Limit          int
}
```

**GetStatsQuery** - 获取统计数据
```go
type GetStatsQuery struct {
    NodeInstanceID uint
    StartTime      time.Time
    EndTime        time.Time
}
```

**GetAlertQuery** - 获取告警配置
```go
type GetAlertQuery struct {
    NodeInstanceID uint
}
```

### 基础设施层

#### PostgreSQL 仓储

```go
type PerformanceRepository struct {
    db *gorm.DB
}

// RecordMetric 记录性能指标
func (r *PerformanceRepository) RecordMetric(ctx context.Context, metric *PerformanceMetric) error {
    return r.db.WithContext(ctx).Create(metric).Error
}

// FindMetrics 查询性能指标
func (r *PerformanceRepository) FindMetrics(ctx context.Context, nodeInstanceID uint, startTime, endTime time.Time, limit int) ([]*PerformanceMetric, error) {
    var metrics []*PerformanceMetric

    query := r.db.WithContext(ctx).
        Where("node_instance_id = ?", nodeInstanceID).
        Where("recorded_at BETWEEN ? AND ?", startTime, endTime).
        Order("recorded_at DESC")

    if limit > 0 {
        query = query.Limit(limit)
    }

    err := query.Find(&metrics).Error
    return metrics, err
}

// GetStats 获取统计数据
func (r *PerformanceRepository) GetStats(ctx context.Context, nodeInstanceID uint, startTime, endTime time.Time) (*PerformanceStats, error) {
    var stats PerformanceStats

    err := r.db.WithContext(ctx).
        Model(&PerformanceMetric{}).
        Select(`
            node_instance_id,
            ? as start_time,
            ? as end_time,
            AVG(cpu_usage) as avg_cpu,
            MAX(cpu_usage) as max_cpu,
            MIN(cpu_usage) as min_cpu,
            AVG(memory_usage) as avg_memory,
            MAX(memory_usage) as max_memory,
            MIN(memory_usage) as min_memory,
            AVG(disk_usage) as avg_disk,
            MAX(disk_usage) as max_disk,
            MIN(disk_usage) as min_disk,
            SUM(network_in) as total_network_in,
            SUM(network_out) as total_network_out
        `, startTime, endTime).
        Where("node_instance_id = ?", nodeInstanceID).
        Where("recorded_at BETWEEN ? AND ?", startTime, endTime).
        Group("node_instance_id").
        Scan(&stats).Error

    if err != nil {
        return nil, err
    }

    return &stats, nil
}
```

## 使用示例

### 1. 记录性能指标

```go
// 节点上报性能指标
cmd := commands.RecordMetricCommand{
    NodeInstanceID: 1,
    CPUUsage:       45.5,
    MemoryUsage:    68.2,
    DiskUsage:      55.0,
    NetworkIn:      1024000,
    NetworkOut:     2048000,
}

handler := commands.NewRecordMetricHandler(repo)
err := handler.Handle(ctx, cmd)
```

### 2. 创建告警配置

```go
// 为节点配置性能告警
cmd := commands.CreateAlertCommand{
    NodeInstanceID:  1,
    Enabled:         true,
    CPUThreshold:    80.0,    // CPU 超过 80% 告警
    MemoryThreshold: 85.0,    // 内存超过 85% 告警
    DiskThreshold:   90.0,    // 磁盘超过 90% 告警
    NetworkInLimit:  10000000,  // 入站流量超过 10MB 告警
    NetworkOutLimit: 20000000,  // 出站流量超过 20MB 告警
}

handler := commands.NewCreateAlertHandler(repo)
alert, err := handler.Handle(ctx, cmd)
```

### 3. 查询性能指标

```go
// 查询最近 1 小时的性能指标
query := queries.GetMetricsQuery{
    NodeInstanceID: 1,
    StartTime:      time.Now().Add(-1 * time.Hour),
    EndTime:        time.Now(),
    Limit:          100,
}

handler := queries.NewGetMetricsHandler(repo)
metrics, err := handler.Handle(ctx, query)
```

### 4. 获取统计数据

```go
// 获取最近 24 小时的统计数据
query := queries.GetStatsQuery{
    NodeInstanceID: 1,
    StartTime:      time.Now().Add(-24 * time.Hour),
    EndTime:        time.Now(),
}

handler := queries.NewGetStatsHandler(repo)
stats, err := handler.Handle(ctx, query)

fmt.Printf("平均 CPU: %.2f%%\n", stats.AvgCPU)
fmt.Printf("最大 CPU: %.2f%%\n", stats.MaxCPU)
fmt.Printf("平均内存: %.2f%%\n", stats.AvgMemory)
```

### 5. 检查告警

```go
// 记录指标时检查是否触发告警
metric := &PerformanceMetric{
    NodeInstanceID: 1,
    CPUUsage:       85.0,
    MemoryUsage:    90.0,
    DiskUsage:      75.0,
}

alert, _ := repo.FindAlertByNode(ctx, metric.NodeInstanceID)
if alert != nil {
    triggeredAlerts := alert.CheckThresholds(metric)
    if len(triggeredAlerts) > 0 {
        // 发送告警通知
        for _, alertType := range triggeredAlerts {
            sendAlert(metric.NodeInstanceID, alertType)
        }
    }
}
```

## API 接口

详见 [节点性能 API 文档](../api/NODE_PERFORMANCE_API.md)

## 数据库表结构

```sql
-- 性能指标表
CREATE TABLE performance_metrics (
    id BIGSERIAL PRIMARY KEY,
    node_instance_id INT NOT NULL,
    cpu_usage DECIMAL(5,2) NOT NULL,
    memory_usage DECIMAL(5,2) NOT NULL,
    disk_usage DECIMAL(5,2) NOT NULL,
    network_in BIGINT NOT NULL DEFAULT 0,
    network_out BIGINT NOT NULL DEFAULT 0,
    recorded_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 性能告警配置表
CREATE TABLE performance_alerts (
    id SERIAL PRIMARY KEY,
    node_instance_id INT NOT NULL UNIQUE,
    enabled BOOLEAN DEFAULT true,
    cpu_threshold DECIMAL(5,2) DEFAULT 80.0,
    memory_threshold DECIMAL(5,2) DEFAULT 85.0,
    disk_threshold DECIMAL(5,2) DEFAULT 90.0,
    network_in_limit BIGINT DEFAULT 0,
    network_out_limit BIGINT DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 索引
CREATE INDEX idx_performance_metrics_node ON performance_metrics(node_instance_id);
CREATE INDEX idx_performance_metrics_time ON performance_metrics(recorded_at DESC);
CREATE INDEX idx_performance_metrics_node_time ON performance_metrics(node_instance_id, recorded_at DESC);
```

## 性能优化

### 1. 时序数据优化

使用 TimescaleDB 超表优化时序数据存储：

```sql
-- 转换为超表
SELECT create_hypertable('performance_metrics', 'recorded_at',
    chunk_time_interval => INTERVAL '1 day');

-- 自动压缩
ALTER TABLE performance_metrics SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'node_instance_id'
);

-- 压缩策略（压缩 7 天前的数据）
SELECT add_compression_policy('performance_metrics', INTERVAL '7 days');

-- 保留策略（删除 90 天前的数据）
SELECT add_retention_policy('performance_metrics', INTERVAL '90 days');
```

### 2. 批量插入

```go
// 批量记录性能指标
func (r *PerformanceRepository) BatchRecordMetrics(ctx context.Context, metrics []*PerformanceMetric) error {
    return r.db.WithContext(ctx).CreateInBatches(metrics, 100).Error
}
```

### 3. 聚合查询优化

```sql
-- 创建物化视图（每小时聚合）
CREATE MATERIALIZED VIEW performance_metrics_hourly AS
SELECT
    node_instance_id,
    date_trunc('hour', recorded_at) as hour,
    AVG(cpu_usage) as avg_cpu,
    MAX(cpu_usage) as max_cpu,
    AVG(memory_usage) as avg_memory,
    MAX(memory_usage) as max_memory,
    AVG(disk_usage) as avg_disk,
    MAX(disk_usage) as max_disk,
    SUM(network_in) as total_network_in,
    SUM(network_out) as total_network_out
FROM performance_metrics
GROUP BY node_instance_id, date_trunc('hour', recorded_at);

-- 创建索引
CREATE INDEX idx_performance_hourly_node_hour
ON performance_metrics_hourly(node_instance_id, hour DESC);

-- 定时刷新（每小时）
CREATE OR REPLACE FUNCTION refresh_performance_hourly()
RETURNS void AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY performance_metrics_hourly;
END;
$$ LANGUAGE plpgsql;
```

## 监控指标说明

### CPU 使用率
- **范围**: 0-100%
- **告警阈值建议**: 80%
- **说明**: 节点 CPU 的平均使用率

### 内存使用率
- **范围**: 0-100%
- **告警阈值建议**: 85%
- **说明**: 节点内存的使用百分比

### 磁盘使用率
- **范围**: 0-100%
- **告警阈值建议**: 90%
- **说明**: 节点磁盘的使用百分比

### 网络流量
- **单位**: bytes
- **说明**:
  - NetworkIn: 入站流量（下载）
  - NetworkOut: 出站流量（上传）

## 扩展功能

### 1. 趋势预测

基于历史数据预测未来趋势：

```go
func PredictTrend(metrics []*PerformanceMetric) *TrendPrediction {
    // 使用线性回归预测未来 1 小时的 CPU 使用率
    // 实现略
}
```

### 2. 异常检测

检测性能指标异常：

```go
func DetectAnomaly(metrics []*PerformanceMetric) []Anomaly {
    // 使用统计方法检测异常值
    // 例如：3-sigma 规则
    // 实现略
}
```

### 3. 性能评分

综合评估节点性能：

```go
func CalculatePerformanceScore(stats *PerformanceStats) float64 {
    // CPU 权重 40%
    cpuScore := (100 - stats.AvgCPU) * 0.4

    // 内存权重 30%
    memoryScore := (100 - stats.AvgMemory) * 0.3

    // 磁盘权重 30%
    diskScore := (100 - stats.AvgDisk) * 0.3

    return cpuScore + memoryScore + diskScore
}
```

## 测试

```go
func TestPerformanceAlert_CheckThresholds(t *testing.T) {
    alert := &PerformanceAlert{
        Enabled:         true,
        CPUThreshold:    80.0,
        MemoryThreshold: 85.0,
        DiskThreshold:   90.0,
    }

    metric := &PerformanceMetric{
        CPUUsage:    85.0,  // 超过阈值
        MemoryUsage: 90.0,  // 超过阈值
        DiskUsage:   75.0,  // 未超过阈值
    }

    alerts := alert.CheckThresholds(metric)

    assert.Len(t, alerts, 2)
    assert.Contains(t, alerts, "cpu")
    assert.Contains(t, alerts, "memory")
}
```

## 最佳实践

### 1. 采集频率

- **生产环境**: 每 1-5 分钟采集一次
- **开发环境**: 每 10-30 分钟采集一次
- **高负载节点**: 可增加采集频率

### 2. 数据保留

- **原始数据**: 保留 7-30 天
- **小时聚合**: 保留 90 天
- **天聚合**: 保留 1 年

### 3. 告警策略

- 设置合理的阈值，避免告警疲劳
- 使用告警静默期，避免重复告警
- 结合多个指标综合判断

## 相关文档

- [节点性能 API 文档](../api/NODE_PERFORMANCE_API.md)
- [节点自动化模块](./NODE_AUTOMATION.md)
- [DDD 架构文档](../architecture/DDD_ARCHITECTURE.md)
