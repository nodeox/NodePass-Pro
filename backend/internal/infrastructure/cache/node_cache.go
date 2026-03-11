package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
	
	"github.com/redis/go-redis/v9"
)

// NodeCache 节点缓存
type NodeCache struct {
	client *redis.Client
	prefix string
	ttl    time.Duration
}

// NewNodeCache 创建节点缓存
func NewNodeCache(client *redis.Client) *NodeCache {
	return &NodeCache{
		client: client,
		prefix: "node:",
		ttl:    5 * time.Minute,
	}
}

// SetOnline 设置节点在线状态（带 TTL）
func (c *NodeCache) SetOnline(ctx context.Context, nodeID string, ttl time.Duration) error {
	key := fmt.Sprintf("%sonline:%s", c.prefix, nodeID)
	return c.client.Set(ctx, key, "1", ttl).Err()
}

// IsOnline 检查节点是否在线
func (c *NodeCache) IsOnline(ctx context.Context, nodeID string) (bool, error) {
	key := fmt.Sprintf("%sonline:%s", c.prefix, nodeID)
	exists, err := c.client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return exists > 0, nil
}

// GetAllOnlineNodes 获取所有在线节点
func (c *NodeCache) GetAllOnlineNodes(ctx context.Context) ([]string, error) {
	pattern := fmt.Sprintf("%sonline:*", c.prefix)
	var nodes []string
	
	iter := c.client.Scan(ctx, 0, pattern, 100).Iterator()
	for iter.Next(ctx) {
		key := iter.Val()
		// 提取节点 ID
		nodeID := key[len(c.prefix)+len("online:"):]
		nodes = append(nodes, nodeID)
	}
	
	if err := iter.Err(); err != nil {
		return nil, err
	}
	
	return nodes, nil
}

// SetNodeInfo 设置节点信息缓存
func (c *NodeCache) SetNodeInfo(ctx context.Context, nodeID string, data map[string]interface{}) error {
	key := fmt.Sprintf("%sinfo:%s", c.prefix, nodeID)
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, key, jsonData, c.ttl).Err()
}

// GetNodeInfo 获取节点信息缓存
func (c *NodeCache) GetNodeInfo(ctx context.Context, nodeID string) (map[string]interface{}, error) {
	key := fmt.Sprintf("%sinfo:%s", c.prefix, nodeID)
	data, err := c.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	
	var info map[string]interface{}
	if err := json.Unmarshal([]byte(data), &info); err != nil {
		return nil, err
	}
	return info, nil
}

// DeleteNodeInfo 删除节点信息缓存
func (c *NodeCache) DeleteNodeInfo(ctx context.Context, nodeID string) error {
	key := fmt.Sprintf("%sinfo:%s", c.prefix, nodeID)
	return c.client.Del(ctx, key).Err()
}

// IncrementGroupNodeCount 增加节点组节点数（原子操作）
func (c *NodeCache) IncrementGroupNodeCount(ctx context.Context, groupID uint, delta int) error {
	key := fmt.Sprintf("%sgroup:%d:count", c.prefix, groupID)
	return c.client.IncrBy(ctx, key, int64(delta)).Err()
}

// GetGroupNodeCount 获取节点组节点数
func (c *NodeCache) GetGroupNodeCount(ctx context.Context, groupID uint) (int, error) {
	key := fmt.Sprintf("%sgroup:%d:count", c.prefix, groupID)
	count, err := c.client.Get(ctx, key).Int()
	if err == redis.Nil {
		return 0, nil
	}
	return count, err
}

// SetGroupOnlineCount 设置节点组在线节点数
func (c *NodeCache) SetGroupOnlineCount(ctx context.Context, groupID uint, count int) error {
	key := fmt.Sprintf("%sgroup:%d:online", c.prefix, groupID)
	return c.client.Set(ctx, key, count, 5*time.Minute).Err()
}

// GetGroupOnlineCount 获取节点组在线节点数
func (c *NodeCache) GetGroupOnlineCount(ctx context.Context, groupID uint) (int, error) {
	key := fmt.Sprintf("%sgroup:%d:online", c.prefix, groupID)
	count, err := c.client.Get(ctx, key).Int()
	if err == redis.Nil {
		return 0, nil
	}
	return count, err
}

// SetNodeMetrics 设置节点指标（用于实时监控）
func (c *NodeCache) SetNodeMetrics(ctx context.Context, nodeID string, metrics map[string]float64) error {
	key := fmt.Sprintf("%smetrics:%s", c.prefix, nodeID)
	
	pipe := c.client.Pipeline()
	for metric, value := range metrics {
		pipe.HSet(ctx, key, metric, value)
	}
	pipe.Expire(ctx, key, 10*time.Minute)
	
	_, err := pipe.Exec(ctx)
	return err
}

// GetNodeMetrics 获取节点指标
func (c *NodeCache) GetNodeMetrics(ctx context.Context, nodeID string) (map[string]string, error) {
	key := fmt.Sprintf("%smetrics:%s", c.prefix, nodeID)
	return c.client.HGetAll(ctx, key).Result()
}

// BatchSetOnline 批量设置节点在线状态
func (c *NodeCache) BatchSetOnline(ctx context.Context, nodeIDs []string, ttl time.Duration) error {
	if len(nodeIDs) == 0 {
		return nil
	}
	
	pipe := c.client.Pipeline()
	for _, nodeID := range nodeIDs {
		key := fmt.Sprintf("%sonline:%s", c.prefix, nodeID)
		pipe.Set(ctx, key, "1", ttl)
	}
	
	_, err := pipe.Exec(ctx)
	return err
}
