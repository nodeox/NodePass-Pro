package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
	
	"github.com/redis/go-redis/v9"
)

// HeartbeatBuffer 心跳缓冲区（Redis 暂存，批量写入 PostgreSQL）
type HeartbeatBuffer struct {
	client *redis.Client
	prefix string
}

// HeartbeatData 心跳数据
type HeartbeatData struct {
	NodeID       string    `json:"node_id"`
	Status       string    `json:"status"`
	CPUUsage     float64   `json:"cpu_usage"`
	MemoryUsage  float64   `json:"memory_usage"`
	TrafficIn    int64     `json:"traffic_in"`
	TrafficOut   int64     `json:"traffic_out"`
	Timestamp    time.Time `json:"timestamp"`
}

// NewHeartbeatBuffer 创建心跳缓冲区
func NewHeartbeatBuffer(client *redis.Client) *HeartbeatBuffer {
	return &HeartbeatBuffer{
		client: client,
		prefix: "heartbeat:buffer:",
	}
}

// Push 推送心跳数据到缓冲区
func (b *HeartbeatBuffer) Push(ctx context.Context, data *HeartbeatData) error {
	key := fmt.Sprintf("%s%s", b.prefix, data.NodeID)
	
	// 序列化
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}
	
	// 推入列表（左侧推入，右侧弹出，FIFO）
	return b.client.LPush(ctx, key, jsonData).Err()
}

// PopBatch 批量弹出心跳数据
func (b *HeartbeatBuffer) PopBatch(ctx context.Context, nodeID string, count int) ([]*HeartbeatData, error) {
	key := fmt.Sprintf("%s%s", b.prefix, nodeID)
	
	// 批量弹出
	results, err := b.client.RPopCount(ctx, key, count).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	
	// 反序列化
	data := make([]*HeartbeatData, 0, len(results))
	for _, result := range results {
		var hb HeartbeatData
		if err := json.Unmarshal([]byte(result), &hb); err != nil {
			continue // 跳过损坏的数据
		}
		data = append(data, &hb)
	}
	
	return data, nil
}

// GetBufferSize 获取缓冲区大小
func (b *HeartbeatBuffer) GetBufferSize(ctx context.Context, nodeID string) (int64, error) {
	key := fmt.Sprintf("%s%s", b.prefix, nodeID)
	return b.client.LLen(ctx, key).Result()
}

// SetNodeOnline 设置节点在线状态（带过期时间）
func (b *HeartbeatBuffer) SetNodeOnline(ctx context.Context, nodeID string, ttl time.Duration) error {
	key := fmt.Sprintf("node:online:%s", nodeID)
	return b.client.Set(ctx, key, "1", ttl).Err()
}

// IsNodeOnline 检查节点是否在线
func (b *HeartbeatBuffer) IsNodeOnline(ctx context.Context, nodeID string) (bool, error) {
	key := fmt.Sprintf("node:online:%s", nodeID)
	exists, err := b.client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return exists > 0, nil
}

// GetAllOnlineNodes 获取所有在线节点
func (b *HeartbeatBuffer) GetAllOnlineNodes(ctx context.Context) ([]string, error) {
	pattern := "node:online:*"
	var nodes []string
	
	iter := b.client.Scan(ctx, 0, pattern, 100).Iterator()
	for iter.Next(ctx) {
		key := iter.Val()
		// 提取节点 ID（去掉前缀）
		nodeID := key[len("node:online:"):]
		nodes = append(nodes, nodeID)
	}
	
	if err := iter.Err(); err != nil {
		return nil, err
	}
	
	return nodes, nil
}
