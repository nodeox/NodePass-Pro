package container

import (
	// 用户模块
	userCommands "nodepass-pro/backend/internal/application/user/commands"
	userQueries "nodepass-pro/backend/internal/application/user/queries"

	// 认证模块
	authCommands "nodepass-pro/backend/internal/application/auth/commands"
	authQueries "nodepass-pro/backend/internal/application/auth/queries"

	// VIP 模块
	vipCommands "nodepass-pro/backend/internal/application/vip/commands"
	vipQueries "nodepass-pro/backend/internal/application/vip/queries"

	// 节点模块
	nodeCommands "nodepass-pro/backend/internal/application/node/commands"
	nodeQueries "nodepass-pro/backend/internal/application/node/queries"

	// 节点组模块
	nodegroupCommands "nodepass-pro/backend/internal/application/nodegroup/commands"
	nodegroupQueries "nodepass-pro/backend/internal/application/nodegroup/queries"

	// 隧道模块
	tunnelCommands "nodepass-pro/backend/internal/application/tunnel/commands"
	tunnelQueries "nodepass-pro/backend/internal/application/tunnel/queries"

	// 流量模块
	trafficCommands "nodepass-pro/backend/internal/application/traffic/commands"
	trafficQueries "nodepass-pro/backend/internal/application/traffic/queries"

	// 权益码模块
	benefitcodeCommands "nodepass-pro/backend/internal/application/benefitcode/commands"
	benefitcodeQueries "nodepass-pro/backend/internal/application/benefitcode/queries"

	// 领域层
	"nodepass-pro/backend/internal/domain/auth"
	"nodepass-pro/backend/internal/domain/benefitcode"
	"nodepass-pro/backend/internal/domain/node"
	"nodepass-pro/backend/internal/domain/nodegroup"
	"nodepass-pro/backend/internal/domain/traffic"
	"nodepass-pro/backend/internal/domain/tunnel"
	"nodepass-pro/backend/internal/domain/user"
	"nodepass-pro/backend/internal/domain/vip"

	// 基础设施层
	"nodepass-pro/backend/internal/infrastructure/cache"
	"nodepass-pro/backend/internal/infrastructure/persistence/postgres"
	authRepo "nodepass-pro/backend/internal/infrastructure/persistence/postgres/auth"
	vipRepo "nodepass-pro/backend/internal/infrastructure/persistence/postgres/vip"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// Container 依赖注入容器
type Container struct {
	// 数据库连接
	DB          *gorm.DB
	RedisClient *redis.Client
	
	// ==================== 仓储层 ====================
	AuthRepo          auth.Repository
	VIPRepo           vip.Repository
	UserRepo          user.Repository
	NodeInstanceRepo  node.InstanceRepository
	NodeGroupRepo     nodegroup.Repository
	TunnelRepo        tunnel.Repository
	TrafficRecordRepo traffic.RecordRepository
	BenefitCodeRepo   benefitcode.Repository

	// ==================== 缓存层 ====================
	AuthCache        *cache.AuthCache
	VIPCache         *cache.VIPCache
	UserCache        *cache.UserCache
	NodeCache        *cache.NodeCache
	NodeGroupCache   *cache.NodeGroupCache
	TunnelCache      *cache.TunnelCache
	TrafficCounter   *cache.TrafficCounter
	HeartbeatBuffer  *cache.HeartbeatBuffer
	BenefitCodeCache *cache.BenefitCodeCache
	
	// ==================== 认证模块 ====================
	// 命令
	LoginHandler           *authCommands.LoginHandler
	RegisterHandler        *authCommands.RegisterHandler
	ChangePasswordHandler  *authCommands.ChangePasswordHandler
	RefreshTokenHandler    *authCommands.RefreshTokenHandler

	// 查询
	GetUserHandler2 *authQueries.GetUserHandler

	// ==================== VIP 模块 ====================
	// 命令
	CreateLevelHandler  *vipCommands.CreateLevelHandler
	UpgradeUserHandler  *vipCommands.UpgradeUserHandler

	// 查询
	ListLevelsHandler *vipQueries.ListLevelsHandler
	GetMyLevelHandler *vipQueries.GetMyLevelHandler

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

	// ==================== 节点组模块 ====================
	// 命令
	CreateGroupHandler *nodegroupCommands.CreateGroupHandler
	UpdateGroupHandler *nodegroupCommands.UpdateGroupHandler
	DeleteGroupHandler *nodegroupCommands.DeleteGroupHandler
	EnableGroupHandler *nodegroupCommands.EnableGroupHandler

	// 查询
	GetGroupHandler      *nodegroupQueries.GetGroupHandler
	ListGroupsHandler    *nodegroupQueries.ListGroupsHandler
	GetGroupStatsHandler *nodegroupQueries.GetGroupStatsHandler

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
	GetUserTrafficHandler    *trafficQueries.GetUserTrafficHandler
	GetTunnelTrafficHandler2 *trafficQueries.GetTunnelTrafficHandler

	// ==================== 权益码模块 ====================
	// 命令
	GenerateCodesHandler *benefitcodeCommands.GenerateCodesHandler
	RedeemCodeHandler    *benefitcodeCommands.RedeemCodeHandler
	RevokeCodeHandler    *benefitcodeCommands.RevokeCodeHandler
	DeleteCodesHandler   *benefitcodeCommands.DeleteCodesHandler

	// 查询
	GetCodeHandler      *benefitcodeQueries.GetCodeHandler
	ListCodesHandler    *benefitcodeQueries.ListCodesHandler
	ValidateCodeHandler *benefitcodeQueries.ValidateCodeHandler
	GetStatsHandler     *benefitcodeQueries.GetStatsHandler
}

// NewContainer 创建依赖注入容器
func NewContainer(db *gorm.DB, redisClient *redis.Client) *Container {
	// ==================== 初始化仓储层 ====================
	authRepo := authRepo.NewAuthRepository(db)
	vipRepo := vipRepo.NewVIPRepository(db)
	userRepo := postgres.NewUserRepository(db)
	nodeInstanceRepo := postgres.NewNodeInstanceRepository(db)
	nodeGroupRepo := postgres.NewNodeGroupRepository(db)
	tunnelRepo := postgres.NewTunnelRepository(db)
	trafficRecordRepo := postgres.NewTrafficRecordRepository(db)
	benefitCodeRepo := postgres.NewBenefitCodeRepository(db)

	// ==================== 初始化缓存层 ====================
	authCache := cache.NewAuthCache(redisClient)
	vipCache := cache.NewVIPCache(redisClient)
	userCache := cache.NewUserCache(redisClient)
	nodeCache := cache.NewNodeCache(redisClient)
	nodeGroupCache := cache.NewNodeGroupCache(redisClient)
	tunnelCache := cache.NewTunnelCache(redisClient)
	trafficCounter := cache.NewTrafficCounter(redisClient)
	heartbeatBuffer := cache.NewHeartbeatBuffer(redisClient)
	benefitCodeCache := cache.NewBenefitCodeCache(redisClient)

	// ==================== 初始化认证模块 ====================
	loginHandler := authCommands.NewLoginHandler(authRepo, userCache)
	registerHandler := authCommands.NewRegisterHandler(authRepo, vipRepo, userCache)
	changePasswordHandler := authCommands.NewChangePasswordHandler(authRepo, userCache)
	refreshTokenHandler := authCommands.NewRefreshTokenHandler(authRepo, userCache)
	getUserHandler2 := authQueries.NewGetUserHandler(authRepo, userCache)

	// ==================== 初始化 VIP 模块 ====================
	createLevelHandler := vipCommands.NewCreateLevelHandler(vipRepo)
	upgradeUserHandler := vipCommands.NewUpgradeUserHandler(vipRepo)
	listLevelsHandler := vipQueries.NewListLevelsHandler(vipRepo)
	getMyLevelHandler := vipQueries.NewGetMyLevelHandler(vipRepo)

	// ==================== 初始化用户模块 ====================
	createUserHandler := userCommands.NewCreateUserHandler(userRepo, userCache)
	getUserHandler := userQueries.NewGetUserHandler(userRepo, userCache)
	
	// ==================== 初始化节点模块 ====================
	heartbeatHandler := nodeCommands.NewHeartbeatHandler(nodeInstanceRepo, nodeCache, heartbeatBuffer)
	getNodeHandler := nodeQueries.NewGetNodeHandler(nodeInstanceRepo, nodeCache)
	listNodesHandler := nodeQueries.NewListNodesHandler(nodeInstanceRepo, nodeCache)
	getOnlineNodesHandler := nodeQueries.NewGetOnlineNodesHandler(nodeInstanceRepo, nodeCache)

	// ==================== 初始化节点组模块 ====================
	createGroupHandler := nodegroupCommands.NewCreateGroupHandler(nodeGroupRepo)
	updateGroupHandler := nodegroupCommands.NewUpdateGroupHandler(nodeGroupRepo)
	deleteGroupHandler := nodegroupCommands.NewDeleteGroupHandler(nodeGroupRepo)
	enableGroupHandler := nodegroupCommands.NewEnableGroupHandler(nodeGroupRepo)

	getGroupHandler := nodegroupQueries.NewGetGroupHandler(nodeGroupRepo)
	listGroupsHandler := nodegroupQueries.NewListGroupsHandler(nodeGroupRepo)
	getGroupStatsHandler := nodegroupQueries.NewGetGroupStatsHandler(nodeGroupRepo)

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

	// ==================== 初始化权益码模块 ====================
	generateCodesHandler := benefitcodeCommands.NewGenerateCodesHandler(benefitCodeRepo)
	redeemCodeHandler := benefitcodeCommands.NewRedeemCodeHandler(benefitCodeRepo)
	revokeCodeHandler := benefitcodeCommands.NewRevokeCodeHandler(benefitCodeRepo)
	deleteCodesHandler := benefitcodeCommands.NewDeleteCodesHandler(benefitCodeRepo)

	getCodeHandler := benefitcodeQueries.NewGetCodeHandler(benefitCodeRepo)
	listCodesHandler := benefitcodeQueries.NewListCodesHandler(benefitCodeRepo)
	validateCodeHandler := benefitcodeQueries.NewValidateCodeHandler(benefitCodeRepo)
	getStatsHandler := benefitcodeQueries.NewGetStatsHandler(benefitCodeRepo)

	return &Container{
		DB:                       db,
		RedisClient:              redisClient,
		AuthRepo:                 authRepo,
		VIPRepo:                  vipRepo,
		UserRepo:                 userRepo,
		NodeInstanceRepo:         nodeInstanceRepo,
		NodeGroupRepo:            nodeGroupRepo,
		TunnelRepo:               tunnelRepo,
		TrafficRecordRepo:        trafficRecordRepo,
		BenefitCodeRepo:          benefitCodeRepo,
		AuthCache:                authCache,
		VIPCache:                 vipCache,
		UserCache:                userCache,
		NodeCache:                nodeCache,
		NodeGroupCache:           nodeGroupCache,
		TunnelCache:              tunnelCache,
		TrafficCounter:           trafficCounter,
		HeartbeatBuffer:          heartbeatBuffer,
		BenefitCodeCache:         benefitCodeCache,
		LoginHandler:             loginHandler,
		RegisterHandler:          registerHandler,
		ChangePasswordHandler:    changePasswordHandler,
		RefreshTokenHandler:      refreshTokenHandler,
		GetUserHandler2:          getUserHandler2,
		CreateLevelHandler:       createLevelHandler,
		UpgradeUserHandler:       upgradeUserHandler,
		ListLevelsHandler:        listLevelsHandler,
		GetMyLevelHandler:        getMyLevelHandler,
		CreateUserHandler:        createUserHandler,
		GetUserHandler:           getUserHandler,
		HeartbeatHandler:         heartbeatHandler,
		GetNodeHandler:           getNodeHandler,
		ListNodesHandler:         listNodesHandler,
		GetOnlineNodesHandler:    getOnlineNodesHandler,
		CreateGroupHandler:       createGroupHandler,
		UpdateGroupHandler:       updateGroupHandler,
		DeleteGroupHandler:       deleteGroupHandler,
		EnableGroupHandler:       enableGroupHandler,
		GetGroupHandler:          getGroupHandler,
		ListGroupsHandler:        listGroupsHandler,
		GetGroupStatsHandler:     getGroupStatsHandler,
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
		GenerateCodesHandler:     generateCodesHandler,
		RedeemCodeHandler:        redeemCodeHandler,
		RevokeCodeHandler:        revokeCodeHandler,
		DeleteCodesHandler:       deleteCodesHandler,
		GetCodeHandler:           getCodeHandler,
		ListCodesHandler:         listCodesHandler,
		ValidateCodeHandler:      validateCodeHandler,
		GetStatsHandler:          getStatsHandler,
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
