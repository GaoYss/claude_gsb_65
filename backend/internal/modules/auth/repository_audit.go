package auth

import (
	"context"
	"fmt"
	"strings"

	"gorm.io/gorm"
)

// AuditFilter 审计日志查询条件。
type AuditFilter struct {
	Actor    string // 用户名 / 姓名关键字
	Action   string
	Resource string
	Result   string
	Keyword  string
	From     string // RFC3339 / 日期时间
	To       string
}

// AuditRepository 审计日志数据访问。
//
// 重要约定: 该仓储【只暴露 Create 与只读查询】, 刻意不提供 Update / Delete /
// Save 方法, 从应用层保证审计记录只追加。数据库侧另由 InstallAuditGuard
// 创建触发器, 在存储层再次拒绝任何 UPDATE / DELETE。
type AuditRepository struct {
	db *gorm.DB
}

// NewAuditRepository 构造审计仓储。
func NewAuditRepository(db *gorm.DB) *AuditRepository {
	return &AuditRepository{db: db}
}

// Append 追加一条审计记录。只插入, 永不更新。
func (r *AuditRepository) Append(ctx context.Context, entry *AuditLog) error {
	if err := r.db.WithContext(ctx).Create(entry).Error; err != nil {
		return fmt.Errorf("写入审计日志失败: %w", err)
	}
	return nil
}

// List 分页查询审计记录, 始终按时间倒序返回。
func (r *AuditRepository) List(ctx context.Context, filter AuditFilter, offset, limit int) ([]AuditLog, int64, error) {
	statement := r.applyFilter(r.db.WithContext(ctx).Model(&AuditLog{}), filter)

	var total int64
	if err := statement.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计审计日志失败: %w", err)
	}

	entries := make([]AuditLog, 0)
	if err := statement.
		Order("occurred_at DESC, id DESC").
		Offset(offset).Limit(limit).
		Find(&entries).Error; err != nil {
		return nil, 0, fmt.Errorf("查询审计日志失败: %w", err)
	}
	return entries, total, nil
}

func (r *AuditRepository) applyFilter(statement *gorm.DB, filter AuditFilter) *gorm.DB {
	if value := strings.TrimSpace(filter.Keyword); value != "" {
		like := "%" + value + "%"
		statement = statement.Where(
			"actor_username LIKE ? OR actor_name LIKE ? OR description LIKE ? OR resource_id LIKE ?",
			like, like, like, like,
		)
	}
	if value := strings.TrimSpace(filter.Actor); value != "" {
		like := "%" + value + "%"
		statement = statement.Where("actor_username LIKE ? OR actor_name LIKE ?", like, like)
	}
	if value := strings.TrimSpace(filter.Action); value != "" {
		statement = statement.Where("action = ?", value)
	}
	if value := strings.TrimSpace(filter.Resource); value != "" {
		statement = statement.Where("resource = ?", value)
	}
	if value := strings.TrimSpace(filter.Result); value != "" {
		statement = statement.Where("result = ?", value)
	}
	if value := strings.TrimSpace(filter.From); value != "" {
		statement = statement.Where("occurred_at >= ?", value)
	}
	if value := strings.TrimSpace(filter.To); value != "" {
		statement = statement.Where("occurred_at <= ?", value)
	}
	return statement
}

// InstallAuditGuard 在数据库层创建触发器, 拒绝任何对审计表的 UPDATE / DELETE。
// 即使有人绕过应用直接连库, 历史操作记录也无法被改写或删除。
func InstallAuditGuard(db *gorm.DB) error {
	dialectorName := db.Dialector.Name()
	switch dialectorName {
	case "sqlite":
		statements := []string{
			`CREATE TRIGGER IF NOT EXISTS trg_audit_no_update
BEFORE UPDATE ON audit_log
BEGIN
	SELECT RAISE(ABORT, 'audit_log 是只追加表, 禁止 UPDATE');
END;`,
			`CREATE TRIGGER IF NOT EXISTS trg_audit_no_delete
BEFORE DELETE ON audit_log
BEGIN
	SELECT RAISE(ABORT, 'audit_log 是只追加表, 禁止 DELETE');
END;`,
		}
		for _, stmt := range statements {
			if err := db.Exec(stmt).Error; err != nil {
				return fmt.Errorf("安装审计保护触发器失败: %w", err)
			}
		}
	case "postgres":
		stmt := `
CREATE OR REPLACE FUNCTION streetlight_block_audit_mutation() RETURNS trigger AS $$
BEGIN
	RAISE EXCEPTION 'audit_log 是只追加表, 禁止 %', TG_OP;
END;
$$ LANGUAGE plpgsql;`
		if err := db.Exec(stmt).Error; err != nil {
			return fmt.Errorf("安装审计保护函数失败: %w", err)
		}
		for _, op := range []string{"UPDATE", "DELETE"} {
			guard := fmt.Sprintf(
				"DROP TRIGGER IF EXISTS trg_audit_no_%s ON audit_log; CREATE TRIGGER trg_audit_no_%s BEFORE %s ON audit_log FOR EACH ROW EXECUTE FUNCTION streetlight_block_audit_mutation();",
				strings.ToLower(op), strings.ToLower(op), op,
			)
			if err := db.Exec(guard).Error; err != nil {
				return fmt.Errorf("安装审计保护触发器失败: %w", err)
			}
		}
	}
	return nil
}
