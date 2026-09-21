package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"
)

// UserRepository 负责用户数据访问。
type UserRepository struct {
	db *gorm.DB
}

// NewUserRepository 构造用户仓储。
func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) session(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx)
}

// Create 新增用户。
func (r *UserRepository) Create(ctx context.Context, user *User) error {
	if err := r.session(ctx).Create(user).Error; err != nil {
		return fmt.Errorf("创建用户失败: %w", err)
	}
	return nil
}

// Save 保存用户全部字段。
func (r *UserRepository) Save(ctx context.Context, user *User) error {
	if err := r.session(ctx).Save(user).Error; err != nil {
		return fmt.Errorf("更新用户失败: %w", err)
	}
	return nil
}

// UpdateColumns 局部更新用户字段(用于改角色/停用)。
func (r *UserRepository) UpdateColumns(ctx context.Context, id uint, columns map[string]any) error {
	result := r.session(ctx).Model(&User{}).Where("id = ?", id).Updates(columns)
	if result.Error != nil {
		return fmt.Errorf("更新用户失败: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrUserNotFound
	}
	return nil
}

// GetByID 按主键查询用户。
func (r *UserRepository) GetByID(ctx context.Context, id uint) (*User, error) {
	var user User
	err := r.session(ctx).First(&user, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("查询用户失败: %w", err)
	}
	return &user, nil
}

// GetByUsername 按用户名查询(含停用用户, 由服务层判断)。
func (r *UserRepository) GetByUsername(ctx context.Context, username string) (*User, error) {
	var user User
	err := r.session(ctx).Where("username = ?", strings.TrimSpace(username)).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("查询用户失败: %w", err)
	}
	return &user, nil
}

// ExistsByUsername 判断用户名是否已存在, excludeID 用于编辑时排除自身。
func (r *UserRepository) ExistsByUsername(ctx context.Context, username string, excludeID uint) (bool, error) {
	var count int64
	statement := r.session(ctx).Model(&User{}).Where("username = ?", strings.TrimSpace(username))
	if excludeID > 0 {
		statement = statement.Where("id <> ?", excludeID)
	}
	if err := statement.Count(&count).Error; err != nil {
		return false, fmt.Errorf("校验用户名失败: %w", err)
	}
	return count > 0, nil
}

// UserFilter 用户列表查询条件。
type UserFilter struct {
	Keyword string
	Role    string
	Active  *bool
}

// List 分页查询用户, 返回记录与总数。
func (r *UserRepository) List(ctx context.Context, filter UserFilter, offset, limit int) ([]User, int64, error) {
	base := func() *gorm.DB {
		statement := r.session(ctx).Model(&User{})
		if keyword := strings.TrimSpace(filter.Keyword); keyword != "" {
			like := "%" + keyword + "%"
			statement = statement.Where("username LIKE ? OR display_name LIKE ? OR team LIKE ?", like, like, like)
		}
		if filter.Role != "" {
			statement = statement.Where("role = ?", filter.Role)
		}
		if filter.Active != nil {
			statement = statement.Where("active = ?", *filter.Active)
		}
		return statement
	}

	var total int64
	if err := base().Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计用户失败: %w", err)
	}

	users := make([]User, 0)
	if err := base().Order("id ASC").Offset(offset).Limit(limit).Find(&users).Error; err != nil {
		return nil, 0, fmt.Errorf("查询用户列表失败: %w", err)
	}
	return users, total, nil
}

// CountAdmins 统计管理岗用户数量, 用于禁止停用最后一个管理员。
func (r *UserRepository) CountAdmins(ctx context.Context, activeOnly bool) (int64, error) {
	var count int64
	statement := r.session(ctx).Model(&User{}).Where("role = ?", RoleManager)
	if activeOnly {
		statement = statement.Where("active = ?", true)
	}
	if err := statement.Count(&count).Error; err != nil {
		return 0, fmt.Errorf("统计管理员失败: %w", err)
	}
	return count, nil
}

// SessionRepository 负责登录会话数据访问。
type SessionRepository struct {
	db *gorm.DB
}

// NewSessionRepository 构造会话仓储。
func NewSessionRepository(db *gorm.DB) *SessionRepository {
	return &SessionRepository{db: db}
}

func (r *SessionRepository) session(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx)
}

// Create 写入会话。
func (r *SessionRepository) Create(ctx context.Context, session *Session) error {
	if err := r.session(ctx).Create(session).Error; err != nil {
		return fmt.Errorf("创建会话失败: %w", err)
	}
	return nil
}

// GetByHash 按令牌摘要查询会话。
func (r *SessionRepository) GetByHash(ctx context.Context, tokenHash string) (*Session, error) {
	var session Session
	err := r.session(ctx).Where("token_hash = ?", tokenHash).First(&session).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("查询会话失败: %w", err)
	}
	return &session, nil
}

// DeleteByHash 注销指定会话。
func (r *SessionRepository) DeleteByHash(ctx context.Context, tokenHash string) error {
	if err := r.session(ctx).Where("token_hash = ?", tokenHash).Delete(&Session{}).Error; err != nil {
		return fmt.Errorf("注销会话失败: %w", err)
	}
	return nil
}

// DeleteByUser 注销某用户的全部会话(用于停用用户或管理员强制下线)。
func (r *SessionRepository) DeleteByUser(ctx context.Context, userID uint) error {
	if err := r.session(ctx).Where("user_id = ?", userID).Delete(&Session{}).Error; err != nil {
		return fmt.Errorf("注销用户会话失败: %w", err)
	}
	return nil
}
