package bootstrap

import (
	"gorm.io/gorm"

	"streetlight/internal/module"
	"streetlight/internal/modules/auth"
	"streetlight/internal/modules/fault"
	"streetlight/internal/modules/lamp"
	"streetlight/internal/modules/repair"
	"streetlight/internal/modules/status"
)

// builtModules 持有装配结果, 认证模块需在路由层作为全局中间件先行挂载。
type builtModules struct {
	all  []module.Module
	auth *auth.Module
}

// buildModules 按依赖方向装配业务模块。
//
// 依赖关系: 路灯台账 <- 故障登记 <- 维修记录, 维修状态查询依赖三者的只读仓储。
// 其中 "删除路灯前校验未闭环故障" 需要路灯模块反向调用故障模块,
// 因此通过 SetOpenFaultCounter 在构造完成后回填, 避免循环构造依赖。
//
// 认证授权(auth)横切所有模块: 业务模块通过 auth.RequirePermission 在路由层声明权限点,
// 认证/审计全局中间件由 bootstrap 统一挂载, 因此 auth 需要最先返回给装配层。
func buildModules(db *gorm.DB) builtModules {
	authModule := auth.New(db)

	lampModule := lamp.New(db)

	faultModule := fault.New(db, lampModule.Service())
	lampModule.Service().SetOpenFaultCounter(faultModule.Repository())

	repairModule := repair.New(db, faultModule.Service())

	statusModule := status.New(
		db,
		lampModule.Repository(),
		faultModule.Repository(),
		repairModule.Repository(),
	)

	return builtModules{
		auth: authModule,
		all: []module.Module{
			authModule,
			lampModule,
			faultModule,
			repairModule,
			statusModule,
		},
	}
}
