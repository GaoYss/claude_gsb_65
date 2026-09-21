package bootstrap

import (
	"context"
	"time"

	"gorm.io/gorm"

	"streetlight/internal/config"
	"streetlight/internal/module"
	"streetlight/internal/modules/auth"
	"streetlight/internal/modules/export"
	"streetlight/internal/modules/fault"
	"streetlight/internal/modules/lamp"
	"streetlight/internal/modules/repair"
	"streetlight/internal/modules/status"
)

// buildModules 按依赖方向装配业务模块。
//
// 认证模块(auth)最先构造, 其 Guard 作为统一鉴权入口注入所有业务模块。
// 依赖关系: 路灯台账 <- 故障登记 <- 维修记录, 维修状态查询依赖三者的只读仓储;
// 数据导出复用故障/维修的列表服务, 因此与页面、接口走同一套数据范围。
func buildModules(db *gorm.DB, cfg config.AuthConfig) ([]module.Module, *auth.Module) {
	ttl := cfg.TokenTTL
	if ttl <= 0 {
		ttl = 12 * time.Hour
	}
	authModule := auth.New(db, cfg.TokenSecret, ttl)

	lampModule := lamp.New(db, authModule.Guard())

	faultModule := fault.New(db, lampModule.Service(), authModule.Guard())
	lampModule.Service().SetOpenFaultCounter(faultModule.Repository())
	faultModule.Service().SetUserDirectory(faultUserDirectory{svc: authModule.Service()})

	repairModule := repair.New(db, faultModule.Service(), authModule.Guard())
	repairModule.Service().SetUserDirectory(repairUserDirectory{svc: authModule.Service()})

	statusModule := status.New(
		db,
		lampModule.Repository(),
		faultModule.Repository(),
		repairModule.Repository(),
		authModule.Guard(),
	)

	exportModule := export.New(faultModule.Service(), repairModule.Service(), authModule.Guard())

	modules := []module.Module{
		authModule,
		lampModule,
		faultModule,
		repairModule,
		statusModule,
		exportModule,
	}
	return modules, authModule
}

// faultUserDirectory 把 auth.Service 适配为故障模块需要的用户目录端口。
type faultUserDirectory struct {
	svc *auth.Service
}

func (d faultUserDirectory) GetAssignee(ctx context.Context, id uint) (fault.AssigneeInfo, error) {
	brief, err := d.svc.GetUserBrief(ctx, id)
	if err != nil {
		return fault.AssigneeInfo{}, err
	}
	return fault.AssigneeInfo{
		ID:          brief.ID,
		Username:    brief.Username,
		DisplayName: brief.DisplayName,
		Role:        brief.Role,
		Active:      brief.Active,
	}, nil
}

// repairUserDirectory 把 auth.Service 适配为维修模块需要的用户目录端口。
type repairUserDirectory struct {
	svc *auth.Service
}

func (d repairUserDirectory) GetAssignee(ctx context.Context, id uint) (repair.AssigneeInfo, error) {
	brief, err := d.svc.GetUserBrief(ctx, id)
	if err != nil {
		return repair.AssigneeInfo{}, err
	}
	return repair.AssigneeInfo{
		ID:          brief.ID,
		Username:    brief.Username,
		DisplayName: brief.DisplayName,
		Role:        brief.Role,
		Active:      brief.Active,
	}, nil
}
