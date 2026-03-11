package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"nodepass-pro/backend/internal/domain/nodegroup"
)

// NodeGroupCache 节点组缓存
type NodeGroupCache struct {
	client *redis.Client
	prefix string
	ttl    time.Duration
}

// NewNodeGroupCache 创建缓存实例
func NewNodeGroupCache(client *redis.Client) *NodeGroupCache {
	return &NodeGroupCache{
		client: client,
		prefix: "nodegroup:",
		ttl:    30 * time.Minute,
	}
}

// SetGroup 缓存节点组
func (c *NodeGroupCache) SetGroup(ctx context.Context, group *nodegroup.NodeGroup) error {
	key := c.groupKey(group.ID)
	data, err := json.Marshal(group)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, key, data, c.ttl).Err()
}

// GetGroup 获取节点组缓存
func (c *NodeGroupCache) GetGroup(ctx context.Context, id uint) (*nodegroup.NodeGroup, error) {
	key := c.groupKey(id)
	data, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // 缓存未命中
		}
		return nil, err
	}

	var group nodegroup.NodeGroup
	if err := json.Unmarshal(data, &group); err != nil {
		return nil, err
	}
	return &group, nil
}

// DeleteGroup 删除节点组缓存
func (c *NodeGroupCache) DeleteGroup(ctx context.Context, id uint) error {
	key := c.groupKey(id)
	return c.client.Del(ctx, key).Err()
}

// SetGroupList 缓存节点组列表
func (c *NodeGroupCache) SetGroupList(ctx context.Context, userID uint, groupType nodegroup.NodeGroupType, groups []*nodegroup.NodeGroup) error {
	key := c.groupListKey(userID, groupType)
	data, err := json.Marshal(groups)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, key, data, c.ttl).Err()
}

// GetGroupList 获取节点组列表缓存
func (c *NodeGroupCache) GetGroupList(ctx context.Context, userID uint, groupType nodegroup.NodeGroupType) ([]*nodegroup.NodeGroup, error) {
	key := c.groupListKey(userID, groupType)
	data, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // 缓存未命中
		}
		return nil, err
	}

	var groups []*nodegroup.NodeGroup
	if err := json.Unmarshal(data, &groups); err != nil {
		return nil, err
	}
	return groups, nil
}

// DeleteGroupList 删除节点组列表缓存
func (c *NodeGroupCache) DeleteGroupList(ctx context.Context, userID uint, groupType nodegroup.NodeGroupType) error {
	key := c.groupListKey(userID, groupType)
	return c.client.Del(ctx, key).Err()
}

// DeleteUserGroupLists 删除用户的所有节点组列表缓存
func (c *NodeGroupCache) DeleteUserGroupLists(ctx context.Context, userID uint) error {
	pattern := fmt.Sprintf("%slist:user:%d:*", c.prefix, userID)
	keys, err := c.scanKeys(ctx, pattern)
	if err != nil {
		return err
	}
	if len(keys) > 0 {
		return c.client.Del(ctx, keys...).Err()
	}
	return nil
}

// SetStats 缓存节点组统计
func (c *NodeGroupCache) SetStats(ctx context.Context, stats *nodegroup.NodeGroupStats) error {
	key := c.statsKey(stats.NodeGroupID)
	data, err := json.Marshal(stats)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, key, data, 5*time.Minute).Err() // 统计数据 TTL 较短
}

// GetStats 获取节点组统计缓存
func (c *NodeGroupCache) GetStats(ctx context.Context, groupID uint) (*nodegroup.NodeGroupStats, error) {
	key := c.statsKey(groupID)
	data, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // 缓存未命中
		}
		return nil, err
	}

	var stats nodegroup.NodeGroupStats
	if err := json.Unmarshal(data, &stats); err != nil {
		return nil, err
	}
	return &stats, nil
}

// DeleteStats 删除节点组统计缓存
func (c *NodeGroupCache) DeleteStats(ctx context.Context, groupID uint) error {
	key := c.statsKey(groupID)
	return c.client.Del(ctx, key).Err()
}

// IncrementNodeCount 增加节点数量
func (c *NodeGroupCache) IncrementNodeCount(ctx context.Context, groupID uint, delta int) error {
	key := c.nodeCountKey(groupID)
	return c.client.IncrBy(ctx, key, int64(delta)).Err()
}

// GetNodeCount 获取节点数量
func (c *NodeGroupCache) GetNodeCount(ctx context.Context, groupID uint) (int, error) {
	key := c.nodeCountKey(groupID)
	count, err := c.client.Get(ctx, key).Int()
	if err != nil {
		if err == redis.Nil {
			return 0, nil
		}
		return 0, err
	}
	return count, nil
}

// SetNodeCount 设置节点数量
func (c *NodeGroupCache) SetNodeCount(ctx context.Context, groupID uint, count int) error {
	key := c.nodeCountKey(groupID)
	return c.client.Set(ctx, key, count, 10*time.Minute).Err()
}

// groupKey 节点组缓存键
func (c *NodeGroupCache) groupKey(id uint) string {
	return fmt.Sprintf("%sgroup:%d", c.prefix, id)
}

// groupListKey 节点组列表缓存键
func (c *NodeGroupCache) groupListKey(userID uint, groupType nodegroup.NodeGroupType) string {
	if groupType == "" {
		return fmt.Sprintf("%slist:user:%d:all", c.prefix, userID)
	}
	return fmt.Sprintf("%slist:user:%d:type:%s", c.prefix, userID, groupType)
}

// statsKey 统计缓存键
func (c *NodeGroupCache) statsKey(groupID uint) string {
	return fmt.Sprintf("%sstats:%d", c.prefix, groupID)
}

// nodeCountKey 节点数量缓存键
func (c *NodeGroupCache) nodeCountKey(groupID uint) string {
	return fmt.Sprintf("%snode_count:%d", c.prefix, groupID)
}

// scanKeys 扫描匹配的键
func (c *NodeGroupCache) scanKeys(ctx context.Context, pattern string) ([]string, error) {
	var keys []string
	iter := c.client.Scan(ctx, 0, pattern, 100).Iterator()
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}
	return keys, iter.Err()
}
