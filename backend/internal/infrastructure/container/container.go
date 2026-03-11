package container

import (
	// 用户模块
	userCommands "nodepass-pro/backend/internal/application/user/commands"
	userQueries "nodepass-pro/backend/internal/application/user/queries"
	
	// 节点模块
	nodeCommands "nodepass-pro/backend/internal/application/node/commands"
	nodeQueries "nodepass-pro/backend/internal/application/node/queries"
	
	// 隧道模块
	tunnelCommands "nodepass-pro/backend/internal/application/tunnel/commands"
	tunnelQueries "nodepass-pro/backend/internal/application/tunnel/queries"
	
	// 流量模块
	trafficCommands "nodepass-pro/backend/internal/application/traffic/commands"
	trafficQueries "nodepass-pro/backend/internal/application/traffic/queries"
	
	// 领域层
	"nodepass-pro/backend/internal/domain/node"
	"nodepass-pro/backend/internal/domain/traffic"
	"nodepass-pro/backend/internal/domain/tunnel"
	"nodepass-pro/backend/internal/domain/user"
	
	// 基础设施层
	"nodepass-pro/backend/internal/infrastructure/cache"
	"nodepass-pro/backend/internal/infrastructure/persistence/postgres"
	
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// Container 依赖注入容器
type Container struct {
	// 数据库连接
	DB          *gorm.DB
	RedisClient *redis.Client
	
	// ==================== 仓储层 ====================
	UserRepo          user.Repository
	NodeInstanceRepo  node.InstanceRepository
	TunnelRepo        tunnel.Repository
	TrafficRecordRepo traffic.RecordRepository
	
	// ==================== 缓存层 ====================
	UserCache       *cache.UserCache
	NodeCache       *cache.NodeCache
	TunnelCache     *cache.TunnelCache
	TrafficCounter  *cache.TrafficCounter
	HeartbeatBuffer *cache.HeartbeatBuffer
	
	// ==================== 用户模块 ====================
	// 命令
	CreateUserHandler *userCommands.CreateUserHandler
	
	// 查询
	GetUserHandler *userQueries.GetUserHandler
	
	// ==================== 节点模块 ====================
	// 命令
	HeartbeatHandler *nodeCommands.HeartbeatHandler
	
	// 查询
	GetNodeHandler        *nodeQueries.GetNodeHandler
	ListNodesHandler      *nodeQueries.ListNodesHandler
	GetOnlineNodesHandler *nodeQueries.GetOnlineNodesHandler
	
	// ==================== 隧道模块 ====================
	// 命令
	CreateTunnelHandler *tunnelCommands.CreateTunnelHandler
	StartTunnelHandler  *tunnelCommands.StartTunnelHandler
	StopTunnelHandler   *tunnelCommands.StopTunnelHandler
	
	// 查询
	GetTunnelHandler         *tunnelQueries.GetTunnelHandler
	ListTunnelsHandler       *tunnelQueries.ListTunnelsHandler
	GetTunnelTrafficHandler  *tunnelQueries.GetTunnelTrafficHandler
	
	// ==================== 流量模块 ====================
	// 命令
	RecordTrafficHandler *trafficCommands.RecordTrafficHandler
	FlushTrafficHandler  *trafficCommands.FlushTrafficHandler
	
	// 查询
	GetUserTrafficHandler   *trafficQueries.GetUserTrafficHandler
	GetTunnelTrafficHandler2 *trafficQueries.GetTunnelTrafficHandler
}

// NewContainer 创建依赖注入容器
func NewContainer(db *gorm.DB, redisClient *redis.Client) *Container {
	// ==================== 初始化仓储层 ====================
	userRepo := postgres.NewUserRepository(db)
	nodeInstanceRepo := postgres.NewNodeInstanceRepository(db)
	tunnelRepo := postgres.NewTunnelRepository(db)
	trafficRecordRepo := postgres.NewTrafficRecordRepository(db)
	
	// ==================== 初始化缓存层 ====================
	userCache := cache.NewUserCache(redisClient)
	nodeCache := cache.NewNodeCache(redisClient)
	tunnelCache := cache.NewTunnelCache(redisClient)
	trafficCounter := cache.NewTrafficCounter(redisClient)
	heartbeatBuffer := cache.NewHeartbeatBuffer(redisClient)
	
	// ==================== 初始化用户模块 ====================
	createUserHandler := userCommands.NewCreateUserHandler(userRepo, userCache)
	getUserHandler := userQueries.NewGetUserHandler(userRepo, userCache)
	
	// ==================== 初始化节点模块 ====================
	heartbeatHandler := nodeCommands.NewHeartbeatHandler(nodeInstanceRepo, nodeCache, heartbeatBuffer)
	getNodeHandler := nodeQueries.NewGetNodeHandler(nodeInstanceRepo, nodeCache)
	listNodesHandler := nodeQueries.NewListNodesHandler(nodeInstanceRepo, nodeCache)
	getOnlineNodesHandler := nodeQueries.NewGetOnlineNodesHandler(nodeInstanceRepo, nodeCache)
	
	// ==================== 初始化隧道模块 ====================
	createTunnelHandler := tunnelCommands.NewCreateTunnelHandler(tunnelRepo, userRepo, tunnelCache)
	startTunnelHandler := tunnelCommands.NewStartTunnelHandler(tunnelRepo, tunnelCache)
	stopTunnelHandler := tunnelCommands.NewStopTunnelHandler(tunnelRepo, tunnelCache)
	
	getTunnelHandler := tunnelQueries.NewGetTunnelHandler(tunnelRepo, tunnelCache)
	listTunnelsHandler := tunnelQueries.NewListTunnelsHandler(tunnelRepo, tunnelCache)
	getTunnelTrafficHandler := tunnelQueries.NewGetTunnelTrafficHandler(tunnelRepo, tunnelCache)
	
	// ==================== 初始化流量模块 ====================
	recordTrafficHandler := trafficCommands.NewRecordTrafficHandler(trafficCounter)
	flushTrafficHandler := trafficCommands.NewFlushTrafficHandler(trafficRecordRepo, trafficCounter)
	
	getUserTrafficHandler := trafficQueries.NewGetUserTrafficHandler(trafficRecordRepo, trafficCounter)
	getTunnelTrafficHandler2 := trafficQueries.NewGetTunnelTrafficHandler(trafficRecordRepo, trafficCounter)
	
	return &Container{
		DB:                       db,
		RedisClient:              redisClient,
		UserRepo:                 userRepo,
		NodeInstanceRepo:         nodeInstanceRepo,
		TunnelRepo:               tunnelRepo,
		TrafficRecordRepo:        trafficRecordRepo,
		UserCache:                userCache,
		NodeCache:                nodeCache,
		TunnelCache:              tunnelCache,
		TrafficCounter:           trafficCounter,
		HeartbeatBuffer:          heartbeatBuffer,
		CreateUserHandler:        createUserHandler,
		GetUserHandler:           getUserHandler,
		HeartbeatHandler:         heartbeatHandler,
		GetNodeHandler:           getNodeHandler,
		ListNodesHandler:         listNodesHandler,
		GetOnlineNodesHandler:    getOnlineNodesHandler,
		CreateTunnelHandler:      createTunnelHandler,
		StartTunnelHandler:       startTunnelHandler,
		StopTunnelHandler:        stopTunnelHandler,
		GetTunnelHandler:         getTunnelHandler,
		ListTunnelsHandler:       listTunnelsHandler,
		GetTunnelTrafficHandler:  getTunnelTrafficHandler,
		RecordTrafficHandler:     recordTrafficHandler,
		FlushTrafficHandler:      flushTrafficHandler,
		GetUserTrafficHandler:    getUserTrafficHandler,
		GetTunnelTrafficHandler2: getTunnelTrafficHandler2,
	}
}

// Close 关闭容器资源
func (c *Container) Close() error {
	// 关闭 Redis 连接
	if c.RedisClient != nil {
		if err := c.RedisClient.Close(); err != nil {
			return err
		}
	}
	
	// 关闭数据库连接
	if c.DB != nil {
		sqlDB, err := c.DB.DB()
		if err != nil {
			return err
		}
		return sqlDB.Close()
	}
	
	return nil
}
