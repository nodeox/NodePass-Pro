# NodePass-Pro 性能监控配置指南

本文档说明如何为 NodePass-Pro 项目添加 Prometheus 指标收集和 OpenTelemetry 分布式追踪。

---

## 📋 目录

1. [Prometheus 指标收集](#prometheus-指标收集)
2. [OpenTelemetry 分布式追踪](#opentelemetry-分布式追踪)
3. [Grafana 可视化](#grafana-可视化)
4. [告警配置](#告警配置)

---

## 🔍 Prometheus 指标收集

### 1. 安装依赖

```bash
cd backend

# 添加 Prometheus 客户端库
go get github.com/prometheus/client_golang/prometheus
go get github.com/prometheus/client_golang/prometheus/promauto
go get github.com/prometheus/client_golang/prometheus/promhttp
```

### 2. 创建指标定义

创建 `backend/internal/metrics/metrics.go`:

```go
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// HTTP 请求指标
	HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nodepass_http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "nodepass_http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path", "status"},
	)

	// 业务指标
	TunnelsTotal = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "nodepass_tunnels_total",
			Help: "Total number of tunnels",
		},
		[]string{"status"},
	)

	NodesTotal = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "nodepass_nodes_total",
			Help: "Total number of nodes",
		},
		[]string{"status", "type"},
	)

	UsersTotal = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "nodepass_users_total",
			Help: "Total number of users",
		},
	)

	TrafficBytesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nodepass_traffic_bytes_total",
			Help: "Total traffic in bytes",
		},
		[]string{"direction"}, // in/out
	)

	// 数据库指标
	DBQueriesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nodepass_db_queries_total",
			Help: "Total number of database queries",
		},
		[]string{"operation", "table"},
	)

	DBQueryDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "nodepass_db_query_duration_seconds",
			Help:    "Database query duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"operation", "table"},
	)

	// Redis 指标
	RedisOperationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nodepass_redis_operations_total",
			Help: "Total number of Redis operations",
		},
		[]string{"operation", "status"},
	)

	// WebSocket 指标
	WebSocketConnectionsActive = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "nodepass_websocket_connections_active",
			Help: "Number of active WebSocket connections",
		},
	)

	WebSocketMessagesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nodepass_websocket_messages_total",
			Help: "Total number of WebSocket messages",
		},
		[]string{"direction"}, // sent/received
	)
)
```

### 3. 创建 Prometheus 中间件

创建 `backend/internal/middleware/prometheus.go`:

```go
package middleware

import (
	"strconv"
	"time"

	"nodepass-pro/backend/internal/metrics"

	"github.com/gin-gonic/gin"
)

// PrometheusMiddleware Prometheus 指标收集中间件
func PrometheusMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// 处理请求
		c.Next()

		// 记录指标
		duration := time.Since(start).Seconds()
		status := strconv.Itoa(c.Writer.Status())
		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}

		metrics.HTTPRequestsTotal.WithLabelValues(
			c.Request.Method,
			path,
			status,
		).Inc()

		metrics.HTTPRequestDuration.WithLabelValues(
			c.Request.Method,
			path,
			status,
		).Observe(duration)
	}
}
```

### 4. 注册 Prometheus 端点

在 `backend/cmd/server/main.go` 中添加:

```go
import (
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func setupRouter(licenseManager *license.Manager) (*gin.Engine, *panelws.Hub) {
	r := gin.New()

	// 添加 Prometheus 中间件
	r.Use(middleware.PrometheusMiddleware())

	// Prometheus 指标端点
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// ... 其他路由
	return r, wsHub
}
```

### 5. Prometheus 配置文件

创建 `prometheus.yml`:

```yaml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

scrape_configs:
  - job_name: 'nodepass-backend'
    static_configs:
      - targets: ['localhost:8080']
    metrics_path: '/metrics'
```

### 6. Docker Compose 集成

在 `docker-compose.yml` 中添加:

```yaml
services:
  prometheus:
    image: prom/prometheus:latest
    container_name: nodepass-prometheus
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml
      - prometheus_data:/prometheus
    command:
      - '--config.file=/etc/prometheus/prometheus.yml'
      - '--storage.tsdb.path=/prometheus'
    ports:
      - "9090:9090"
    networks:
      - nodepass-network

volumes:
  prometheus_data:
```

---

## 🔭 OpenTelemetry 分布式追踪

### 1. 安装依赖

```bash
cd backend

# 添加 OpenTelemetry 依赖
go get go.opentelemetry.io/otel
go get go.opentelemetry.io/otel/trace
go get go.opentelemetry.io/otel/sdk/trace
go get go.opentelemetry.io/otel/exporters/jaeger
go get go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin
```

### 2. 初始化 OpenTelemetry

创建 `backend/internal/telemetry/telemetry.go`:

```go
package telemetry

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
)

// InitTracer 初始化追踪器
func InitTracer(serviceName, jaegerEndpoint string) (*sdktrace.TracerProvider, error) {
	// 创建 Jaeger exporter
	exporter, err := jaeger.New(jaeger.WithCollectorEndpoint(
		jaeger.WithEndpoint(jaegerEndpoint),
	))
	if err != nil {
		return nil, err
	}

	// 创建 TracerProvider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(serviceName),
		)),
	)

	// 设置全局 TracerProvider
	otel.SetTracerProvider(tp)

	return tp, nil
}

// Shutdown 关闭追踪器
func Shutdown(ctx context.Context, tp *sdktrace.TracerProvider) error {
	return tp.Shutdown(ctx)
}
```

### 3. 添加追踪中间件

在 `main.go` 中:

```go
import (
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

func setupRouter(licenseManager *license.Manager) (*gin.Engine, *panelws.Hub) {
	r := gin.New()

	// 添加 OpenTelemetry 中间件
	r.Use(otelgin.Middleware("nodepass-backend"))

	// ... 其他路由
	return r, wsHub
}

func main() {
	// 初始化追踪器
	tp, err := telemetry.InitTracer("nodepass-backend", "http://localhost:14268/api/traces")
	if err != nil {
		zap.L().Fatal("初始化追踪器失败", zap.Error(err))
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := telemetry.Shutdown(ctx, tp); err != nil {
			zap.L().Error("关闭追踪器失败", zap.Error(err))
		}
	}()

	// ... 其他代码
}
```

### 4. 在服务中添加 Span

```go
import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

func (s *TunnelService) Create(ctx context.Context, userID uint, req *CreateTunnelRequest) (*models.Tunnel, error) {
	tracer := otel.Tracer("tunnel-service")
	ctx, span := tracer.Start(ctx, "TunnelService.Create")
	defer span.End()

	span.SetAttributes(
		attribute.Int64("user_id", int64(userID)),
		attribute.String("tunnel_name", req.Name),
	)

	// 业务逻辑...

	return tunnel, nil
}
```

### 5. Jaeger Docker Compose

在 `docker-compose.yml` 中添加:

```yaml
services:
  jaeger:
    image: jaegertracing/all-in-one:latest
    container_name: nodepass-jaeger
    environment:
      - COLLECTOR_ZIPKIN_HOST_PORT=:9411
    ports:
      - "5775:5775/udp"
      - "6831:6831/udp"
      - "6832:6832/udp"
      - "5778:5778"
      - "16686:16686"  # Jaeger UI
      - "14268:14268"  # Jaeger collector
      - "14250:14250"
      - "9411:9411"
    networks:
      - nodepass-network
```

---

## 📊 Grafana 可视化

### 1. Docker Compose 配置

```yaml
services:
  grafana:
    image: grafana/grafana:latest
    container_name: nodepass-grafana
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=admin
      - GF_USERS_ALLOW_SIGN_UP=false
    volumes:
      - grafana_data:/var/lib/grafana
      - ./grafana/dashboards:/etc/grafana/provisioning/dashboards
      - ./grafana/datasources:/etc/grafana/provisioning/datasources
    ports:
      - "3000:3000"
    networks:
      - nodepass-network
    depends_on:
      - prometheus

volumes:
  grafana_data:
```

### 2. 数据源配置

创建 `grafana/datasources/prometheus.yml`:

```yaml
apiVersion: 1

datasources:
  - name: Prometheus
    type: prometheus
    access: proxy
    url: http://prometheus:9090
    isDefault: true
```

### 3. 仪表板配置

创建 `grafana/dashboards/nodepass.json` (示例):

```json
{
  "dashboard": {
    "title": "NodePass Pro Monitoring",
    "panels": [
      {
        "title": "HTTP Requests Rate",
        "targets": [
          {
            "expr": "rate(nodepass_http_requests_total[5m])"
          }
        ]
      },
      {
        "title": "HTTP Request Duration",
        "targets": [
          {
            "expr": "histogram_quantile(0.95, rate(nodepass_http_request_duration_seconds_bucket[5m]))"
          }
        ]
      }
    ]
  }
}
```

---

## 🚨 告警配置

### 1. Prometheus 告警规则

创建 `prometheus/alerts.yml`:

```yaml
groups:
  - name: nodepass_alerts
    interval: 30s
    rules:
      - alert: HighErrorRate
        expr: rate(nodepass_http_requests_total{status=~"5.."}[5m]) > 0.05
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "High error rate detected"
          description: "Error rate is {{ $value }} requests/sec"

      - alert: HighResponseTime
        expr: histogram_quantile(0.95, rate(nodepass_http_request_duration_seconds_bucket[5m])) > 1
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High response time detected"
          description: "95th percentile response time is {{ $value }}s"

      - alert: LowDiskSpace
        expr: node_filesystem_avail_bytes / node_filesystem_size_bytes < 0.1
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Low disk space"
          description: "Disk space is below 10%"
```

### 2. Alertmanager 配置

创建 `alertmanager.yml`:

```yaml
global:
  resolve_timeout: 5m

route:
  group_by: ['alertname']
  group_wait: 10s
  group_interval: 10s
  repeat_interval: 1h
  receiver: 'webhook'

receivers:
  - name: 'webhook'
    webhook_configs:
      - url: 'http://localhost:8080/api/v1/alerts/webhook'
```

---

## 🚀 使用指南

### 启动监控栈

```bash
# 启动所有服务
docker compose up -d

# 查看服务状态
docker compose ps
```

### 访问监控界面

- **Prometheus**: http://localhost:9090
- **Grafana**: http://localhost:3000 (admin/admin)
- **Jaeger UI**: http://localhost:16686

### 查看指标

```bash
# 查看所有指标
curl http://localhost:8080/metrics

# 查看特定指标
curl http://localhost:8080/metrics | grep nodepass_http_requests_total
```

### Makefile 集成

```makefile
.PHONY: monitoring-up
monitoring-up: ## 启动监控服务
	docker compose -f docker-compose.monitoring.yml up -d

.PHONY: monitoring-down
monitoring-down: ## 停止监控服务
	docker compose -f docker-compose.monitoring.yml down

.PHONY: monitoring-logs
monitoring-logs: ## 查看监控日志
	docker compose -f docker-compose.monitoring.yml logs -f
```

---

## 📈 关键指标说明

### HTTP 指标
- `nodepass_http_requests_total` - 总请求数
- `nodepass_http_request_duration_seconds` - 请求耗时

### 业务指标
- `nodepass_tunnels_total` - 隧道总数
- `nodepass_nodes_total` - 节点总数
- `nodepass_users_total` - 用户总数
- `nodepass_traffic_bytes_total` - 流量总量

### 系统指标
- `nodepass_db_queries_total` - 数据库查询数
- `nodepass_db_query_duration_seconds` - 数据库查询耗时
- `nodepass_redis_operations_total` - Redis 操作数
- `nodepass_websocket_connections_active` - 活跃 WebSocket 连接数

---

## ✅ 检查清单

完成性能监控配置后，确保：

- [ ] Prometheus 客户端库已安装
- [ ] 指标定义已创建
- [ ] Prometheus 中间件已添加
- [ ] /metrics 端点可访问
- [ ] Prometheus 服务已启动
- [ ] OpenTelemetry 已初始化
- [ ] Jaeger 服务已启动
- [ ] Grafana 已配置数据源
- [ ] 告警规则已配置
- [ ] 监控仪表板已创建

---

**下一步**: 根据业务需求添加更多自定义指标和告警规则。
