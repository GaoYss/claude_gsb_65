package auth

import (
	"context"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

// AuditRepository 负责审计日志的数据访问。
// 安全约束: 这里只暴露 Create / List, 不提供 Update / Delete,
// 从数据访问层保证审计记录只增不改, 权限变更后历史操作记录无法被隐藏或改写。
type AuditRepository struct {
	db *gorm.DB
}

// NewAuditRepository 构造审计仓储。
func NewAuditRepository(db *gorm.DB) *AuditRepository {
	return &AuditRepository{db: db}
}

func (r *AuditRepository) session(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx)
}

// Create 追加一条审计记录。
func (r *AuditRepository) Create(ctx context.Context, entry *AuditLog) error {
	if err := r.session(ctx).Create(entry).Error; err != nil {
		return fmt.Errorf("写入审计记录失败: %w", err)
	}
	return nil
}

// AuditFilter 审计日志查询条件。
type AuditFilter struct {
	ActorUsername string
	Action        string
	Result        string
	ResourceType  string
	Keyword       string
	From          *time.Time
	To            *time.Time
}

// List 分页查询审计记录, 按时间倒序。
func (r *AuditRepository) List(ctx context.Context, filter AuditFilter, offset, limit int) ([]AuditLog, int64, error) {
	base := func() *gorm.DB {
		statement := r.session(ctx).Model(&AuditLog{})
		if value := strings.TrimSpace(filter.ActorUsername); value != "" {
			statement = statement.Where("actor_username LIKE ?", "%"+value+"%")
		}
		if value := strings.TrimSpace(filter.Action); value != "" {
			statement = statement.Where("action = ?", value)
		}
		if value := strings.TrimSpace(filter.Result); value != "" {
			statement = statement.Where("result = ?", value)
		}
		if value := strings.TrimSpace(filter.ResourceType); value != "" {
			statement = statement.Where("resource_type = ?", value)
		}
		if value := strings.TrimSpace(filter.Keyword); value != "" {
			like := "%" + value + "%"
			statement = statement.Where(
				"resource_no LIKE ? OR path LIKE ? OR detail LIKE ? OR missing_permission LIKE ?",
				like, like, like, like,
			)
		}
		if filter.From != nil {
			statement = statement.Where("created_at >= ?", *filter.From)
		}
		if filter.To != nil {
			statement = statement.Where("created_at < ?", *filter.To)
		}
		return statement
	}

	var total int64
	if err := base().Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计审计记录失败: %w", err)
	}

	entries := make([]AuditLog, 0)
	if err := base().Order("id DESC").Offset(offset).Limit(limit).Find(&entries).Error; err != nil {
		return nil, 0, fmt.Errorf("查询审计记录失败: %w", err)
	}
	return entries, total, nil
}
