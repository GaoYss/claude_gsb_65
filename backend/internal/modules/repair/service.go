package repair

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"streetlight/internal/apperr"
	"streetlight/internal/modules/auth"
	"streetlight/internal/modules/fault"
	"streetlight/pkg/pagination"
)

// repairSortSpec 定义维修记录列表允许的排序字段白名单。
var repairSortSpec = pagination.SortSpec{
	Allowed: map[string]string{
		"repair_no":   "repair_no",
		"started_at":  "started_at",
		"finished_at": "finished_at",
		"status":      "status",
		"result":      "result",
		"cost":        "cost",
		"created_at":  "created_at",
	},
	Default: "started_at",
}

// FaultPort 由故障登记模块实现, 维修模块通过它联动故障状态与路灯状态。
type FaultPort interface {
	GetByID(ctx context.Context, id uint) (*fault.Fault, error)
	OnRepairStarted(ctx context.Context, faultID uint, repairID uint) error
	OnRepairFinished(ctx context.Context, faultID uint, fixed bool) error
	SyncRepairStats(ctx context.Context, faultID uint, repairCount int, latestRepairID *uint) error
}

// AssigneeInfo 维修负责人简要信息。
type AssigneeInfo struct {
	ID          uint
	Username    string
	DisplayName string
	Role        string
	Active      bool
}

// UserDirectoryPort 由认证模块实现, 派工/改派时校验负责人身份。
type UserDirectoryPort interface {
	GetAssignee(ctx context.Context, id uint) (AssigneeInfo, error)
}

// Service 承载维修记录录入的业务规则。
type Service struct {
	repo   *Repository
	faults FaultPort
	users  UserDirectoryPort
}

// NewService 构造维修记录服务。
func NewService(repo *Repository, faults FaultPort) *Service {
	return &Service{repo: repo, faults: faults}
}

// SetUserDirectory 注入用户目录端口, 由 bootstrap 装配以避免构造循环。
func (s *Service) SetUserDirectory(users UserDirectoryPort) { s.users = users }

// Get 查询维修记录详情。
func (s *Service) Get(ctx context.Context, id uint) (*Repair, error) {
	return s.repo.GetByID(ctx, id)
}

// List 分页查询维修记录。
//
// 数据范围(与页面/导出/看板同一套角色结论):
//   - 管理岗(有 repair:manage): 全部记录;
//   - 维修人员(有 repair:advance, 无 manage): 仅自己负责的记录;
//   - 登记人员(只有 repair:view): 全部只读。
func (s *Service) List(ctx context.Context, query ListQuery) ([]Repair, int64, pagination.Query, error) {
	page := pagination.Parse(query.Params, repairSortSpec)
	filter, err := buildFilter(query)
	if err != nil {
		return nil, 0, page, err
	}
	if operator := auth.FromContext(ctx); operator != nil {
		if !operator.Can(auth.PermRepairManage) && operator.Can(auth.PermRepairAdvance) {
			filter.AssigneeID = operator.ID
		}
	}
	items, total, err := s.repo.List(ctx, filter, page)
	if err != nil {
		return nil, 0, page, err
	}
	return items, total, page, nil
}

// ListByFault 查询某条故障的维修过程记录。
func (s *Service) ListByFault(ctx context.Context, faultID uint) ([]Repair, error) {
	if _, err := s.faults.GetByID(ctx, faultID); err != nil {
		return nil, err
	}
	return s.repo.ListByFault(ctx, faultID)
}

// Create 录入维修记录(维修开工), 并联动故障与路灯状态。
//
// 负责人(assignee)判定:
//   - 维修人员自助开工: 只能以本人为负责人, 请求里的 assignee_id 被忽略;
//   - 管理岗派工(有 repair:assign/manage): 可指定任意启用中的维修人员为负责人。
func (s *Service) Create(ctx context.Context, req CreateRequest) (*Repair, error) {
	operator := auth.FromContext(ctx)

	target, err := s.faults.GetByID(ctx, req.FaultID)
	if err != nil {
		return nil, err
	}
	if target.Status == fault.StatusClosed {
		return nil, apperr.Conflict("故障 %s 已关闭, 不允许再登记维修记录", target.FaultNo)
	}
	if target.Status == fault.StatusRepaired {
		return nil, apperr.Conflict("故障 %s 已修复, 如需返修请先登记新的维修记录并重新开工", target.FaultNo)
	}

	ongoing, err := s.repo.GetOngoingByFault(ctx, target.ID)
	if err != nil {
		return nil, err
	}
	if ongoing != nil {
		return nil, apperr.Conflict("故障 %s 已有进行中的维修记录 %s, 请先完成后再录入", target.FaultNo, ongoing.RepairNo)
	}

	// 确定负责人与维修人员署名。
	assigneeID, repairman, err := s.resolveAssignee(ctx, operator, req)
	if err != nil {
		return nil, err
	}

	startedAt, err := parseTime(req.StartedAt, time.Now())
	if err != nil {
		return nil, err
	}
	if startedAt.Before(target.ReportedAt) {
		return nil, apperr.BadRequest("开工时间不能早于故障上报时间 %s", target.ReportedAt.Format("2006-01-02 15:04:05"))
	}

	entity := &Repair{
		FaultID:      target.ID,
		FaultNo:      target.FaultNo,
		LampID:       target.LampID,
		LampCode:     target.LampCode,
		AssigneeID:   assigneeID,
		Repairman:    repairman,
		RepairTeam:   strings.TrimSpace(req.RepairTeam),
		ContactPhone: strings.TrimSpace(req.ContactPhone),
		StartedAt:    startedAt,
		Status:       StatusOngoing,
		Content:      strings.TrimSpace(req.Content),
		Materials:    strings.TrimSpace(req.Materials),
		Cost:         valueOrZero(req.Cost),
		Remark:       strings.TrimSpace(req.Remark),
	}

	if err := s.repo.CreateWithUniqueNo(ctx, entity, "WX"+startedAt.Format("20060102")); err != nil {
		return nil, err
	}

	// 开工后: 故障转为维修中, 路灯转为维修状态
	if err := s.faults.OnRepairStarted(ctx, target.ID, entity.ID); err != nil {
		return nil, err
	}

	entity.FillDuration()
	return entity, nil
}

// resolveAssignee 依据操作者与请求确定维修负责人, 返回 assigneeID 与署名。
func (s *Service) resolveAssignee(ctx context.Context, operator *auth.Principal, req CreateRequest) (uint, string, error) {
	if operator == nil {
		return 0, "", apperr.Unauthenticated("未登录, 无法录入维修记录")
	}

	requestedName := strings.TrimSpace(req.Repairman)

	// 管理岗派工: 可指定负责人。
	if operator.Can(auth.PermRepairAssign) || operator.Can(auth.PermRepairManage) {
		if req.AssigneeID > 0 {
			if s.users == nil {
				return 0, "", apperr.Internal("用户目录未装配, 无法派工")
			}
			target, err := s.users.GetAssignee(ctx, req.AssigneeID)
			if err != nil {
				return 0, "", err
			}
			if !target.Active {
				return 0, "", apperr.BadRequest("目标负责人已停用, 无法派工")
			}
			name := requestedName
			if name == "" {
				name = target.DisplayName
			}
			return target.ID, name, nil
		}
		// 未指定负责人时, 管理岗自己挂名。
		name := requestedName
		if name == "" {
			name = operator.DisplayName
		}
		return operator.ID, name, nil
	}

	// 维修人员: 只能以本人为负责人。
	if req.AssigneeID > 0 && req.AssigneeID != operator.ID {
		return 0, "", apperr.PermissionDenied(
			auth.PermRepairAssign,
			"你不能把维修记录指派给他人, 缺少授权: "+auth.PermRepairAssign+"("+auth.PermissionLabel(auth.PermRepairAssign)+")",
		)
	}
	name := requestedName
	if name == "" {
		name = operator.DisplayName
	}
	return operator.ID, name, nil
}

// Reassign 改派维修负责人(仅管理岗)。
func (s *Service) Reassign(ctx context.Context, id uint, req ReassignRequest) (*Repair, error) {
	entity, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if s.users == nil {
		return nil, apperr.Internal("用户目录未装配, 无法改派")
	}
	target, err := s.users.GetAssignee(ctx, req.AssigneeID)
	if err != nil {
		return nil, err
	}
	if !target.Active {
		return nil, apperr.BadRequest("目标负责人已停用, 无法改派")
	}
	if target.Role != "repair" && target.Role != "admin" {
		return nil, apperr.BadRequest("只能改派给维修人员, 目标角色: %s", target.Role)
	}
	entity.AssigneeID = target.ID
	entity.Repairman = target.DisplayName
	if err := s.repo.Update(ctx, entity); err != nil {
		return nil, err
	}
	entity.FillDuration()
	return entity, nil
}

// Update 修改维修记录, 已完成的记录不允许修改。
func (s *Service) Update(ctx context.Context, id uint, req UpdateRequest) (*Repair, error) {
	entity, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if entity.Status == StatusFinished {
		return nil, apperr.Conflict("维修记录 %s 已完成, 不允许修改", entity.RepairNo)
	}

	if req.Repairman != nil {
		repairman := strings.TrimSpace(*req.Repairman)
		if repairman == "" {
			return nil, apperr.BadRequest("维修人员不能为空")
		}
		entity.Repairman = repairman
	}
	if req.RepairTeam != nil {
		entity.RepairTeam = strings.TrimSpace(*req.RepairTeam)
	}
	if req.ContactPhone != nil {
		entity.ContactPhone = strings.TrimSpace(*req.ContactPhone)
	}
	if req.StartedAt != nil {
		startedAt, err := parseTime(*req.StartedAt, entity.StartedAt)
		if err != nil {
			return nil, err
		}
		entity.StartedAt = startedAt
	}
	if req.Content != nil {
		entity.Content = strings.TrimSpace(*req.Content)
	}
	if req.Materials != nil {
		entity.Materials = strings.TrimSpace(*req.Materials)
	}
	if req.Cost != nil {
		entity.Cost = *req.Cost
	}
	if req.Remark != nil {
		entity.Remark = strings.TrimSpace(*req.Remark)
	}

	if err := s.repo.Update(ctx, entity); err != nil {
		return nil, err
	}
	entity.FillDuration()
	return entity, nil
}

// Finish 完成维修: 记录结果与完工时间, 结果为已修复时联动故障转为已修复。
func (s *Service) Finish(ctx context.Context, id uint, req FinishRequest) (*Repair, error) {
	entity, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if entity.Status == StatusFinished {
		return nil, apperr.Conflict("维修记录 %s 已完成, 不允许重复提交", entity.RepairNo)
	}

	result := strings.TrimSpace(req.Result)
	if !IsValidResult(result) {
		return nil, apperr.BadRequest("非法的维修结果: %s", result)
	}

	finishedAt, err := parseTime(req.FinishedAt, time.Now())
	if err != nil {
		return nil, err
	}
	if finishedAt.Before(entity.StartedAt) {
		return nil, apperr.BadRequest("完工时间不能早于开工时间")
	}

	entity.FinishedAt = &finishedAt
	entity.Status = StatusFinished
	entity.Result = result
	if content := strings.TrimSpace(req.Content); content != "" {
		entity.Content = content
	}
	if materials := strings.TrimSpace(req.Materials); materials != "" {
		entity.Materials = materials
	}
	if req.Cost != nil {
		entity.Cost = *req.Cost
	}
	if remark := strings.TrimSpace(req.Remark); remark != "" {
		entity.Remark = remark
	}

	if err := s.repo.Update(ctx, entity); err != nil {
		return nil, err
	}

	if err := s.faults.OnRepairFinished(ctx, entity.FaultID, result == ResultFixed); err != nil {
		return nil, err
	}

	entity.FillDuration()
	return entity, nil
}

// Delete 删除维修记录, 已关闭故障的维修记录不允许删除。
func (s *Service) Delete(ctx context.Context, id uint) error {
	entity, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	target, err := s.faults.GetByID(ctx, entity.FaultID)
	if err != nil {
		return err
	}
	if target.Status == fault.StatusClosed {
		return apperr.Conflict("故障 %s 已关闭, 不允许删除其维修记录", target.FaultNo)
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}

	count, err := s.repo.CountByFault(ctx, entity.FaultID)
	if err != nil {
		return err
	}
	latest, err := s.repo.LatestByFault(ctx, entity.FaultID)
	if err != nil {
		return err
	}
	var latestID *uint
	if latest != nil {
		latestID = &latest.ID
	}

	if err := s.faults.SyncRepairStats(ctx, entity.FaultID, int(count), latestID); err != nil {
		slog.Warn("同步故障维修统计失败", "fault_id", entity.FaultID, "error", err)
	}
	return nil
}

// Metadata 返回维修模块字典。
func (s *Service) Metadata(ctx context.Context) (*Meta, error) {
	repairmen, err := s.repo.DistinctValues(ctx, "repairman")
	if err != nil {
		return nil, err
	}
	teams, err := s.repo.DistinctValues(ctx, "repair_team")
	if err != nil {
		return nil, err
	}
	return &Meta{
		Statuses:  Statuses(),
		Results:   Results(),
		Repairmen: repairmen,
		Teams:     teams,
	}, nil
}

// Statistics 汇总维修统计信息。
func (s *Service) Statistics(ctx context.Context) (*Statistics, error) {
	total, err := s.repo.Count(ctx)
	if err != nil {
		return nil, err
	}
	byStatus, err := s.repo.CountByColumn(ctx, "status")
	if err != nil {
		return nil, err
	}
	totalCost, err := s.repo.SumCost(ctx)
	if err != nil {
		return nil, err
	}
	averageDuration, err := s.repo.AverageDurationHours(ctx)
	if err != nil {
		return nil, err
	}

	result := &Statistics{
		Total:             total,
		OngoingTotal:      byStatus[StatusOngoing],
		FinishedTotal:     byStatus[StatusFinished],
		TotalCost:         totalCost,
		AverageDurationHr: averageDuration,
	}
	if result.FinishedTotal > 0 {
		result.AverageCost = totalCost / float64(result.FinishedTotal)
	}
	return result, nil
}

// buildFilter 将查询参数转换为仓储条件并解析日期区间。
func buildFilter(query ListQuery) (Filter, error) {
	filter := Filter{
		Keyword:    strings.TrimSpace(query.Keyword),
		FaultID:    query.FaultID,
		LampID:     query.LampID,
		Repairman:  strings.TrimSpace(query.Repairman),
		RepairTeam: strings.TrimSpace(query.RepairTeam),
		Status:     strings.TrimSpace(query.Status),
		Result:     strings.TrimSpace(query.Result),
	}
	if filter.Status != "" && filter.Status != StatusOngoing && filter.Status != StatusFinished {
		return filter, apperr.BadRequest("非法的维修状态: %s", filter.Status)
	}
	if filter.Result != "" && !IsValidResult(filter.Result) {
		return filter, apperr.BadRequest("非法的维修结果: %s", filter.Result)
	}

	if value := strings.TrimSpace(query.StartDate); value != "" {
		from, err := parseDay(value)
		if err != nil {
			return filter, err
		}
		filter.StartedFrom = &from
	}
	if value := strings.TrimSpace(query.EndDate); value != "" {
		to, err := parseDay(value)
		if err != nil {
			return filter, err
		}
		to = to.AddDate(0, 0, 1)
		filter.StartedTo = &to
	}
	if filter.StartedFrom != nil && filter.StartedTo != nil && filter.StartedTo.Before(*filter.StartedFrom) {
		return filter, apperr.BadRequest("结束日期不能早于开始日期")
	}
	return filter, nil
}

// parseTime 解析时间字符串, 为空时返回 fallback。
func parseTime(value string, fallback time.Time) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback, nil
	}
	for _, layout := range []string{time.RFC3339, "2006-01-02 15:04:05", "2006-01-02T15:04:05", "2006-01-02"} {
		if parsed, err := time.ParseInLocation(layout, value, time.Local); err == nil {
			return parsed, nil
		}
	}
	return time.Time{}, apperr.BadRequest("时间格式不正确, 建议使用 YYYY-MM-DD HH:mm:ss: %s", value)
}

// parseDay 解析 YYYY-MM-DD 日期, 返回当天零点。
func parseDay(value string) (time.Time, error) {
	date, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(value), time.Local)
	if err != nil {
		return time.Time{}, apperr.BadRequest("日期格式应为 YYYY-MM-DD, 当前值: %s", value)
	}
	return date, nil
}

func valueOrZero(value *float64) float64 {
	if value == nil {
		return 0
	}
	return *value
}
