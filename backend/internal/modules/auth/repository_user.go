package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"streetlight/internal/apperr"
)

// UserRepository 用户数据访问。
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

// Create 新建用户。
func (r *UserRepository) Create(ctx context.Context, user *User) error {
	if err := r.session(ctx).Create(user).Error; err != nil {
		return fmt.Errorf("创建用户失败: %w", err)
	}
	return nil
}

// Save 全字段保存用户(供重置密码/启停/改角色使用)。
func (r *UserRepository) Save(ctx context.Context, user *User) error {
	if err := r.session(ctx).Save(user).Error; err != nil {
		return fmt.Errorf("保存用户失败: %w", err)
	}
	return nil
}

// GetByID 按主键查询用户。
func (r *UserRepository) GetByID(ctx context.Context, id uint) (*User, error) {
	var user User
	err := r.session(ctx).First(&user, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperr.NotFound("用户不存在: id=%d", id)
	}
	if err != nil {
		return nil, fmt.Errorf("查询用户失败: %w", err)
	}
	return &user, nil
}

// GetByUsername 按用户名查询, 不存在返回 apperr.NotFound。
func (r *UserRepository) GetByUsername(ctx context.Context, username string) (*User, error) {
	var user User
	err := r.session(ctx).Where("username = ?", strings.TrimSpace(username)).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperr.NotFound("用户不存在或密码错误")
	}
	if err != nil {
		return nil, fmt.Errorf("查询用户失败: %w", err)
	}
	return &user, nil
}

// List 分页查询全部用户。
func (r *UserRepository) List(ctx context.Context, keyword, role string) ([]User, int64, error) {
	statement := r.session(ctx).Model(&User{})
	if value := strings.TrimSpace(keyword); value != "" {
		like := "%" + value + "%"
		statement = statement.Where("username LIKE ? OR display_name LIKE ?", like, like)
	}
	if value := strings.TrimSpace(role); value != "" {
		statement = statement.Where("role = ?", value)
	}

	var total int64
	if err := statement.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计用户失败: %w", err)
	}
	users := make([]User, 0)
	if err := statement.Order("id ASC").Find(&users).Error; err != nil {
		return nil, 0, fmt.Errorf("查询用户列表失败: %w", err)
	}
	return users, total, nil
}

// Count 返回用户总数, 用于判断是否需要初始化内置账号。
func (r *UserRepository) Count(ctx context.Context) (int64, error) {
	var total int64
	if err := r.session(ctx).Model(&User{}).Count(&total).Error; err != nil {
		return 0, fmt.Errorf("统计用户失败: %w", err)
	}
	return total, nil
}
