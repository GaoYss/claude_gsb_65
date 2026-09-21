package auth

import (
	"context"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"streetlight/internal/apperr"
)

// Service 处理登录认证与用户/角色管理。
type Service struct {
	users  *UserRepository
	tokens *TokenService
}

// NewService 构造认证服务。
func NewService(users *UserRepository, tokens *TokenService) *Service {
	return &Service{users: users, tokens: tokens}
}

// Login 校验账号密码并签发令牌。
func (s *Service) Login(ctx context.Context, username, password, ip string) (*LoginResult, error) {
	user, err := s.users.GetByUsername(ctx, username)
	if err != nil {
		// 账号不存在与密码错误返回同一提示, 避免账号枚举。
		return nil, apperr.BadRequest("用户名或密码不正确")
	}
	if !user.Active {
		return nil, apperr.PermissionDenied("", "账号已被停用, 请联系管理岗")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, apperr.BadRequest("用户名或密码不正确")
	}

	token, expiresAt, err := s.tokens.Issue(user.ID, user.Username, time.Now())
	if err != nil {
		return nil, apperr.Internal("签发登录令牌失败").WithCause(err)
	}

	return s.buildLoginResult(user, token, expiresAt), nil
}

// Profile 依据已认证用户组装个人信息与权限快照。
func (s *Service) Profile(ctx context.Context, current *Principal) (*LoginResult, error) {
	user, err := s.users.GetByID(ctx, current.ID)
	if err != nil {
		return nil, err
	}
	return s.buildLoginResult(user, "", time.Time{}), nil
}

// buildLoginResult 组装返回给前端的会话信息。permissions 显式展开,
// 是前端页面/按钮/路由显隐的唯一依据。
func (s *Service) buildLoginResult(user *User, token string, expiresAt time.Time) *LoginResult {
	permissions := PermissionsForRole(user.Role)
	return &LoginResult{
		Token:     token,
		ExpiresAt: expiresAt,
		User: UserProfile{
			ID:          user.ID,
			Username:    user.Username,
			DisplayName: user.DisplayName,
			Role:        user.Role,
			RoleLabel:   RoleLabel(user.Role),
			Active:      user.Active,
		},
		Permissions: permissions,
		Roles:       RoleCatalog(),
	}
}

// ListUsers 查询用户列表(管理岗)。
func (s *Service) ListUsers(ctx context.Context, keyword, role string) ([]UserProfile, int64, error) {
	users, total, err := s.users.List(ctx, keyword, role)
	if err != nil {
		return nil, 0, err
	}
	items := make([]UserProfile, 0, len(users))
	for _, user := range users {
		items = append(items, UserProfile{
			ID:          user.ID,
			Username:    user.Username,
			DisplayName: user.DisplayName,
			Role:        user.Role,
			RoleLabel:   RoleLabel(user.Role),
			Active:      user.Active,
		})
	}
	return items, total, nil
}

// UserBrief 是跨模块改派时需要的用户简要信息。
type UserBrief struct {
	ID          uint
	Username    string
	DisplayName string
	Role        string
	Active      bool
}

// GetUserBrief 按 ID 查询启用中的用户, 供改派时校验负责人。
func (s *Service) GetUserBrief(ctx context.Context, id uint) (*UserBrief, error) {
	user, err := s.users.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return &UserBrief{
		ID: user.ID, Username: user.Username, DisplayName: user.DisplayName,
		Role: user.Role, Active: user.Active,
	}, nil
}

// CreateUser 新建用户(管理岗)。
func (s *Service) CreateUser(ctx context.Context, req UserUpsertRequest) (*UserProfile, error) {
	username := strings.TrimSpace(req.Username)
	if len(username) < 3 {
		return nil, apperr.BadRequest("用户名至少 3 个字符")
	}
	if len(req.Password) < 6 {
		return nil, apperr.BadRequest("密码至少 6 位")
	}
	if !IsValidRole(req.Role) {
		return nil, apperr.BadRequest("非法的角色: %s", req.Role)
	}
	if _, err := s.users.GetByUsername(ctx, username); err == nil {
		return nil, apperr.Conflict("用户名已存在: %s", username)
	}

	hash, err := hashPassword(req.Password)
	if err != nil {
		return nil, err
	}
	user := &User{
		Username:     username,
		PasswordHash: hash,
		DisplayName:  strings.TrimSpace(req.DisplayName),
		Role:         req.Role,
		Active:       true,
	}
	if user.DisplayName == "" {
		user.DisplayName = username
	}
	if err := s.users.Create(ctx, user); err != nil {
		return nil, err
	}
	return &UserProfile{
		ID: user.ID, Username: user.Username, DisplayName: user.DisplayName,
		Role: user.Role, RoleLabel: RoleLabel(user.Role), Active: user.Active,
	}, nil
}

// UpdateUser 修改用户角色/显示名/启停状态(管理岗)。
// 调整角色会立即改变该用户的权限判定, 但既有的审计记录因已冗余快照而不受影响。
func (s *Service) UpdateUser(ctx context.Context, id uint, req UserUpdateRequest) (*UserProfile, error) {
	user, err := s.users.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if req.Role != nil {
		if !IsValidRole(*req.Role) {
			return nil, apperr.BadRequest("非法的角色: %s", *req.Role)
		}
		user.Role = *req.Role
	}
	if req.DisplayName != nil {
		name := strings.TrimSpace(*req.DisplayName)
		if name == "" {
			return nil, apperr.BadRequest("显示名不能为空")
		}
		user.DisplayName = name
	}
	if req.Active != nil {
		user.Active = *req.Active
	}
	if err := s.users.Save(ctx, user); err != nil {
		return nil, err
	}
	return &UserProfile{
		ID: user.ID, Username: user.Username, DisplayName: user.DisplayName,
		Role: user.Role, RoleLabel: RoleLabel(user.Role), Active: user.Active,
	}, nil
}

// ResetPassword 重置指定用户密码(管理岗)。
func (s *Service) ResetPassword(ctx context.Context, id uint, password string) error {
	if len(password) < 6 {
		return apperr.BadRequest("新密码至少 6 位")
	}
	user, err := s.users.GetByID(ctx, id)
	if err != nil {
		return err
	}
	hash, err := hashPassword(password)
	if err != nil {
		return err
	}
	user.PasswordHash = hash
	return s.users.Save(ctx, user)
}

// ChangePassword 用户修改自身密码, 需校验原密码。
func (s *Service) ChangePassword(ctx context.Context, current *Principal, oldPassword, newPassword string) error {
	if len(newPassword) < 6 {
		return apperr.BadRequest("新密码至少 6 位")
	}
	user, err := s.users.GetByID(ctx, current.ID)
	if err != nil {
		return err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(oldPassword)); err != nil {
		return apperr.BadRequest("原密码不正确")
	}
	hash, err := hashPassword(newPassword)
	if err != nil {
		return err
	}
	user.PasswordHash = hash
	return s.users.Save(ctx, user)
}

// EnsureSeedUsers 在内置账号缺失时初始化三个演示账号, 每个角色一个。
func (s *Service) EnsureSeedUsers(ctx context.Context) error {
	count, err := s.users.Count(ctx)
	if err != nil || count > 0 {
		return err
	}
	seeds := []struct {
		username string
		password string
		name     string
		role     string
	}{
		{"registrar", "registrar123", "登记员·林晓", RoleRegistrar},
		{"repairman", "repair123", "维修工·刘志强", RoleRepair},
		{"admin", "admin123", "管理员·周敏", RoleAdmin},
	}
	for _, item := range seeds {
		hash, err := hashPassword(item.password)
		if err != nil {
			return err
		}
		user := &User{
			Username: item.username, PasswordHash: hash,
			DisplayName: item.name, Role: item.role, Active: true,
		}
		if err := s.users.Create(ctx, user); err != nil {
			return err
		}
	}
	return nil
}

func hashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", apperr.Internal("密码加密失败")
	}
	return string(hash), nil
}
