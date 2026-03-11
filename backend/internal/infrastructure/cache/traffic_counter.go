package cache

import (
	"context"
	"fmt"
	"time"
	
	"github.com/redis/go-redis/v9"
)

// TrafficCounter 流量计数器（Redis 实时计数）
type TrafficCounter struct {
	client *redis.Client
	prefix string
}

// NewTrafficCounter 创建流量计数器
func NewTrafficCounter(client *redis.Client) *TrafficCounter {
	return &TrafficCounter{
		client: client,
		prefix: "traffic:",
	}
}

// IncrementUserTraffic 增加用户流量（原子操作）
func (c *TrafficCounter) IncrementUserTraffic(ctx context.Context, userID uint, inBytes, outBytes int64) error {
	pipe := c.client.Pipeline()
	
	inKey := fmt.Sprintf("%suser:%d:in", c.prefix, userID)
	outKey := fmt.Sprintf("%suser:%d:out", c.prefix, userID)
	
	pipe.IncrBy(ctx, inKey, inBytes)
	pipe.IncrBy(ctx, outKey, outBytes)
	
	// 设置过期时间（7 天）
	pipe.Expire(ctx, inKey, 7*24*time.Hour)
	pipe.Expire(ctx, outKey, 7*24*time.Hour)
	
	_, err := pipe.Exec(ctx)
	return err
}

// GetUserTraffic 获取用户流量
func (c *TrafficCounter) GetUserTraffic(ctx context.Context, userID uint) (inBytes, outBytes int64, err error) {
	pipe := c.client.Pipeline()
	
	inKey := fmt.Sprintf("%suser:%d:in", c.prefix, userID)
	outKey := fmt.Sprintf("%suser:%d:out", c.prefix, userID)
	
	inCmd := pipe.Get(ctx, inKey)
	outCmd := pipe.Get(ctx, outKey)
	
	if _, err := pipe.Exec(ctx); err != nil && err != redis.Nil {
		return 0, 0, err
	}
	
	inBytes, _ = inCmd.Int64()
	outBytes, _ = outCmd.Int64()
	
	return inBytes, outBytes, nil
}

// ResetUserTraffic 重置用户流量
func (c *TrafficCounter) ResetUserTraffic(ctx context.Context, userID uint) error {
	inKey := fmt.Sprintf("%suser:%d:in", c.prefix, userID)
	outKey := fmt.Sprintf("%suser:%d:out", c.prefix, userID)
	
	pipe := c.client.Pipeline()
	pipe.Del(ctx, inKey)
	pipe.Del(ctx, outKey)
	
	_, err := pipe.Exec(ctx)
	return err
}

// IncrementTunnelTraffic 增加隧道流量
func (c *TrafficCounter) IncrementTunnelTraffic(ctx context.Context, tunnelID uint, inBytes, outBytes int64) error {
	pipe := c.client.Pipeline()
	
	inKey := fmt.Sprintf("%stunnel:%d:in", c.prefix, tunnelID)
	outKey := fmt.Sprintf("%stunnel:%d:out", c.prefix, tunnelID)
	
	pipe.IncrBy(ctx, inKey, inBytes)
	pipe.IncrBy(ctx, outKey, outBytes)
	
	pipe.Expire(ctx, inKey, 7*24*time.Hour)
	pipe.Expire(ctx, outKey, 7*24*time.Hour)
	
	_, err := pipe.Exec(ctx)
	return err
}

// GetTunnelTraffic 获取隧道流量
func (c *TrafficCounter) GetTunnelTraffic(ctx context.Context, tunnelID uint) (inBytes, outBytes int64, err error) {
	pipe := c.client.Pipeline()
	
	inKey := fmt.Sprintf("%stunnel:%d:in", c.prefix, tunnelID)
	outKey := fmt.Sprintf("%stunnel:%d:out", c.prefix, tunnelID)
	
	inCmd := pipe.Get(ctx, inKey)
	outCmd := pipe.Get(ctx, outKey)
	
	if _, err := pipe.Exec(ctx); err != nil && err != redis.Nil {
		return 0, 0, err
	}
	
	inBytes, _ = inCmd.Int64()
	outBytes, _ = outCmd.Int64()
	
	return inBytes, outBytes, nil
}

// BatchGetUserTraffic 批量获取用户流量
func (c *TrafficCounter) BatchGetUserTraffic(ctx context.Context, userIDs []uint) (map[uint]struct{ In, Out int64 }, error) {
	if len(userIDs) == 0 {
		return make(map[uint]struct{ In, Out int64 }), nil
	}
	
	pipe := c.client.Pipeline()
	
	type cmdPair struct {
		userID uint
		inCmd  *redis.StringCmd
		outCmd *redis.StringCmd
	}
	
	cmds := make([]cmdPair, len(userIDs))
	for i, userID := range userIDs {
		inKey := fmt.Sprintf("%suser:%d:in", c.prefix, userID)
		outKey := fmt.Sprintf("%suser:%d:out", c.prefix, userID)
		
		cmds[i] = cmdPair{
			userID: userID,
			inCmd:  pipe.Get(ctx, inKey),
			outCmd: pipe.Get(ctx, outKey),
		}
	}
	
	if _, err := pipe.Exec(ctx); err != nil && err != redis.Nil {
		return nil, err
	}
	
	result := make(map[uint]struct{ In, Out int64 })
	for _, cmd := range cmds {
		inBytes, _ := cmd.inCmd.Int64()
		outBytes, _ := cmd.outCmd.Int64()
		result[cmd.userID] = struct{ In, Out int64 }{In: inBytes, Out: outBytes}
	}
	
	return result, nil
}
