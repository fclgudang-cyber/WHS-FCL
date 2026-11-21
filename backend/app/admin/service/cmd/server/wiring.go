package main

import (
	"github.com/Mikdad-Force/Ghost/app/admin/service/internal/data"
	"github.com/Mikdad-Force/Ghost/app/admin/service/internal/fcl"
	"github.com/Mikdad-Force/Ghost/app/admin/service/internal/server"
	"github.com/Mikdad-Force/Ghost/app/admin/service/internal/service"
	"github.com/Mikdad-Force/Ghost/pkg/middleware/auth"
	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
)

// initApp 手写装配整个应用,是本服务的依赖注入点(替代原 google/wire 生成)。
// initApp assembles the whole application by hand — the dependency injection point.
//
// 装配严格单向分层,自上而下的阅读顺序即依赖方向:
// The wiring is strictly layered; reading top-down follows the dependency direction:
//
//	基础设施 → 服务客户端(data) → 服务层(service) → 传输层(server)
//
// 本服务是 BFF 网关:不直连数据库,仓储层由对 core-service 的 gRPC 服务客户端替代。
// 约定 / Conventions:
//   - 新增模块:先加服务客户端行,再加服务构造行,并传给 NewRestServer;漏接由编译器在调用处报错。
//     To add a module: append a service-client line, then a service constructor, then pass it to
//     NewRestServer; a missing connection is a compile error at the call site.
//   - 持有 cleanup 的资源创建成功后立即注册进 cleanups;任何一步失败,rollback 逆序执行已注册
//     的清理(shutdown 与中途失败共用同一条 LIFO 路径)。
//     Resources owning a cleanup register it immediately; rollback runs them LIFO both on
//     mid-way failure and on shutdown.
//   - 本文件只做构造与传参,不写业务逻辑。
//     Construction and parameter passing only; no business logic in this file.
func initApp(ctx *bootstrap.Context) (*kratos.App, func(), error) {
	// cleanup 注册表:rollback 时逆序执行。
	// Cleanup registry; rollback runs entries in reverse order.
	var cleanups []func()
	rollback := func() {
		for i := len(cleanups) - 1; i >= 0; i-- {
			cleanups[i]()
		}
	}

	// ═══════════════════════ 一、基础设施 ═══════════════════════

	discovery := data.NewDiscovery(ctx)
	authenticationServiceClient := data.NewAuthenticationServiceClient(ctx, discovery)
	clientType := data.NewClientType()
	accessTokenChecker := auth.NewTokenChecker(ctx, authenticationServiceClient, clientType)
	engine := data.NewAuthorizer()
	redisClient, cleanupRedis, err := data.NewRedisClient(ctx)
	if err != nil {
		rollback()
		return nil, nil, err
	}
	cleanups = append(cleanups, cleanupRedis)

	// Backend FCL berjalan di dalam admin-service (tanpa service terpisah).
	// Kegagalan koneksi DB FCL tidak boleh menghentikan service utama.
	if fclServers, ferr := fcl.Start(); ferr != nil {
		log.Errorf("fcl backend disabled: %v", ferr)
	} else {
		cleanups = append(cleanups, func() { fcl.Stop(fclServers) })
	}

	captcha := data.NewCaptcha(redisClient)
	minIOClient := data.NewMinIoClient(ctx)

	// ═══════════════════════ 二、服务客户端(internal/data) ═══════════════════════

	apiAuditLogServiceClient := data.NewApiAuditLogServiceClient(ctx, discovery)
	loginAuditLogServiceClient := data.NewLoginAuditLogServiceClient(ctx, discovery)
	userServiceClient := data.NewUserServiceClient(ctx, discovery)
	tenantServiceClient := data.NewTenantServiceClient(ctx, discovery)
	orgUnitServiceClient := data.NewOrgUnitServiceClient(ctx, discovery)
	positionServiceClient := data.NewPositionServiceClient(ctx, discovery)
	roleServiceClient := data.NewRoleServiceClient(ctx, discovery)
	userCredentialServiceClient := data.NewUserCredentialServiceClient(ctx, discovery)
	menuServiceClient := data.NewMenuServiceClient(ctx, discovery)
	apiServiceClient := data.NewApiServiceClient(ctx, discovery)
	permissionServiceClient := data.NewPermissionServiceClient(ctx, discovery)
	permissionGroupServiceClient := data.NewPermissionGroupServiceClient(ctx, discovery)
	taskServiceClient := data.NewTaskServiceClient(ctx, discovery)
	warehouseServiceClient := data.NewWarehouseServiceClient(ctx, discovery)
	locationServiceClient := data.NewLocationServiceClient(ctx, discovery)
	stockQuantServiceClient := data.NewStockQuantServiceClient(ctx, discovery)
	stockLotServiceClient := data.NewStockLotServiceClient(ctx, discovery)
	stockPickingServiceClient := data.NewStockPickingServiceClient(ctx, discovery)
	purchaseServiceClient := data.NewPurchaseServiceClient(ctx, discovery)
	orderServiceClient := data.NewOrderServiceClient(ctx, discovery)
	pickingTaskServiceClient := data.NewPickingTaskServiceClient(ctx, discovery)
	waveServiceClient := data.NewWaveServiceClient(ctx, discovery)
	settingServiceClient := data.NewSettingServiceClient(ctx, discovery)
	sortingServiceClient := data.NewSortingServiceClient(ctx, discovery)
	approvalRequestServiceClient := data.NewApprovalRequestServiceClient(ctx, discovery)
	approvalFlowServiceClient := data.NewApprovalFlowServiceClient(ctx, discovery)
	productServiceClient := data.NewProductServiceClient(ctx, discovery)
	loginPolicyServiceClient := data.NewLoginPolicyServiceClient(ctx, discovery)
	dictTypeServiceClient := data.NewDictTypeServiceClient(ctx, discovery)
	dictEntryServiceClient := data.NewDictEntryServiceClient(ctx, discovery)
	languageServiceClient := data.NewLanguageServiceClient(ctx, discovery)
	fileServiceClient := data.NewFileServiceClient(ctx, discovery)
	internalMessageServiceClient := data.NewInternalMessageServiceClient(ctx, discovery)
	internalMessageCategoryServiceClient := data.NewInternalMessageCategoryServiceClient(ctx, discovery)
	internalMessageRecipientServiceClient := data.NewInternalMessageRecipientServiceClient(ctx, discovery)
	dataAccessAuditLogServiceClient := data.NewDataAccessAuditLogServiceClient(ctx, discovery)
	policyEvaluationLogServiceClient := data.NewPolicyEvaluationLogServiceClient(ctx, discovery)
	operationAuditLogServiceClient := data.NewOperationAuditLogServiceClient(ctx, discovery)
	permissionAuditLogServiceClient := data.NewPermissionAuditLogServiceClient(ctx, discovery)

	// ── register:client ── 新模块服务客户端在此行后注册(make register 工具锚点,勿删)

	// ═══════════════════════ 三、服务层(internal/service) ═══════════════════════

	userService := service.NewUserService(ctx, userServiceClient, tenantServiceClient, orgUnitServiceClient, positionServiceClient, roleServiceClient, userCredentialServiceClient)
	userProfileService := service.NewUserProfileService(ctx, userServiceClient, tenantServiceClient, orgUnitServiceClient, positionServiceClient, roleServiceClient, userCredentialServiceClient)
	roleService := service.NewRoleService(ctx, roleServiceClient, tenantServiceClient)
	tenantService := service.NewTenantService(ctx, userServiceClient, userCredentialServiceClient, tenantServiceClient, roleServiceClient, captcha)
	orgUnitService := service.NewOrgUnitService(ctx, orgUnitServiceClient, userServiceClient)
	positionService := service.NewPositionService(ctx, positionServiceClient, orgUnitServiceClient)
	menuService := service.NewMenuService(ctx, menuServiceClient)
	apiService := service.NewApiService(ctx, apiServiceClient)
	permissionGroupService := service.NewPermissionGroupService(ctx, permissionServiceClient, permissionGroupServiceClient)
	permissionService := service.NewPermissionService(ctx, permissionServiceClient, permissionGroupServiceClient, roleServiceClient, apiServiceClient, menuServiceClient)
	adminPortalService := service.NewRouterService(ctx, menuServiceClient, permissionServiceClient, roleServiceClient, userServiceClient)
	taskService := service.NewTaskService(ctx, taskServiceClient)
	warehouseService := service.NewWarehouseService(ctx, warehouseServiceClient)
	locationService := service.NewLocationService(ctx, locationServiceClient)
	stockQuantService := service.NewStockQuantService(ctx, stockQuantServiceClient)
	stockLotService := service.NewStockLotService(ctx, stockLotServiceClient)
	stockPickingService := service.NewStockPickingService(ctx, stockPickingServiceClient)
	purchaseService := service.NewPurchaseService(ctx, purchaseServiceClient)
	orderService := service.NewOrderService(ctx, orderServiceClient)
	pickingTaskService := service.NewPickingTaskService(ctx, pickingTaskServiceClient)
	waveService := service.NewWaveService(ctx, waveServiceClient)
	settingService := service.NewSettingService(ctx, settingServiceClient)
	sortingService := service.NewSortingService(ctx, sortingServiceClient)
	approvalRequestService := service.NewApprovalRequestService(ctx, approvalRequestServiceClient)
	approvalFlowService := service.NewApprovalFlowService(ctx, approvalFlowServiceClient)
	productService := service.NewProductService(ctx, productServiceClient)
	authenticationService := service.NewAuthenticationService(ctx, authenticationServiceClient, captcha)
	loginPolicyService := service.NewLoginPolicyService(ctx, loginPolicyServiceClient)
	dictTypeService := service.NewDictTypeService(ctx, dictTypeServiceClient)
	dictEntryService := service.NewDictEntryService(ctx, dictEntryServiceClient)
	languageService := service.NewLanguageService(ctx, languageServiceClient)
	fileService := service.NewFileService(ctx, fileServiceClient)
	fileTransferService := service.NewFileTransferService(ctx, minIOClient, fileServiceClient)
	internalMessageService := service.NewInternalMessageService(ctx, internalMessageServiceClient, internalMessageCategoryServiceClient, internalMessageRecipientServiceClient, authenticationServiceClient, userServiceClient, clientType)
	internalMessageCategoryService := service.NewInternalMessageCategoryService(ctx, internalMessageCategoryServiceClient)
	internalMessageRecipientService := service.NewInternalMessageRecipientService(ctx, internalMessageServiceClient, internalMessageRecipientServiceClient)
	apiAuditLogService := service.NewApiAuditLogService(ctx, apiAuditLogServiceClient, apiServiceClient)
	dataAccessAuditLogService := service.NewDataAccessAuditLogService(ctx, dataAccessAuditLogServiceClient)
	loginAuditLogService := service.NewLoginAuditLogService(ctx, loginAuditLogServiceClient)
	policyEvaluationLogService := service.NewPolicyEvaluationLogService(ctx, policyEvaluationLogServiceClient)
	operationAuditLogService := service.NewOperationAuditLogService(ctx, operationAuditLogServiceClient)
	permissionAuditLogService := service.NewPermissionAuditLogService(ctx, permissionAuditLogServiceClient)

	// ── register:service ── 新模块服务在此行后注册(make register 工具锚点,勿删)

	// ═══════════════════════ 四、传输层(internal/server) ═══════════════════════

	restMiddlewares := server.NewRestMiddleware(ctx, accessTokenChecker, engine, apiAuditLogServiceClient, loginAuditLogServiceClient)

	httpServer := server.NewRestServer(ctx, restMiddlewares,
		userService,
		userProfileService,
		roleService,
		tenantService,
		orgUnitService,
		positionService,
		menuService,
		apiService,
		permissionGroupService,
		permissionService,
		adminPortalService,
		taskService,
		warehouseService,
		locationService,
		stockQuantService,
		stockLotService,
		stockPickingService,
		purchaseService,
		waveService,
		pickingTaskService,
		settingService,
		sortingService,
		orderService,
		approvalRequestService,
		approvalFlowService,
		productService,
		authenticationService,
		loginPolicyService,
		dictTypeService,
		dictEntryService,
		languageService,
		fileService,
		fileTransferService,
		internalMessageService,
		internalMessageCategoryService,
		internalMessageRecipientService,
		apiAuditLogService,
		dataAccessAuditLogService,
		loginAuditLogService,
		policyEvaluationLogService,
		operationAuditLogService,
		permissionAuditLogService,
		// register:rest-arg ── 新模块服务实参在此行后追加(make register 工具锚点,勿删)
	)

	grpcMiddlewares := server.NewGrpcMiddleware(ctx)

	grpcServer, err := server.NewGrpcServer(ctx, grpcMiddlewares)
	if err != nil {
		rollback()
		return nil, nil, err
	}

	sseServer := server.NewSseServer(ctx, internalMessageService)

	return newApp(ctx, httpServer, grpcServer, sseServer), rollback, nil
}
