package auth

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"gorm.io/gorm"

	"streetlight/internal/apperr"
)

// DefaultSessionTTL 是登录会话的默认有效期。
const DefaultSessionTTL = 12 * time.Hour

// Service 承载认证、用户管理与审计写入的业务规则。
type Service struct {
	users    *UserRepository
	sessions *SessionRepository
	audits   *AuditRepository
	ttl      time.Duration
}

// NewService 构造认证服务。
func NewService(db *gorm.DB, ttl time.Duration) *Service {
	return &Service{
		users:    NewUserRepository(db),
		sessions: NewSessionRepository(db),
		audits:   NewAuditRepository(db),
		ttl:      ttl,
	}
}

// LoginResult 登录成功后返回令牌与当前用户信息。
type LoginResult struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	User      *Profile  `json:"user"`
}

// Login 校验用户名密码并签发会话令牌。
func (s *Service) Login(ctx context.Context, username, password string) (*LoginResult, error) {
	user, err := s.users.GetByUsername(ctx, username)
	if err != nil {
		return nil, apperr.New(401, "UNAUTHORIZED", "用户名或密码不正确")
	}
	if !user.Active {
		return nil, apperr.New(403, "FORBIDDEN", ErrUserDisabled.Error())
	}
	if !checkPassword(user.PasswordHash, password) {
		return nil, apperr.New(401, "UNAUTHORIZED", "用户名或密码不正确")
	}

	token, err := generateToken()
	if err != nil {
		return nil, apperr.Internal("生成登录令牌失败").WithCause(err)
	}
	expiresAt := time.Now().Add(s.ttl)
	if err := s.sessions.Create(ctx, &Session{
		TokenHash: hashToken(token),
		UserID:    user.ID,
		ExpiresAt: expiresAt,
	}); err != nil {
		return nil, err
	}

	now := time.Now()
	_ = s.users.UpdateColumns(ctx, user.ID, map[string]any{"last_login_at": &now})

	return &LoginResult{Token: token, ExpiresAt: expiresAt, User: user.Profile()}, nil
}

// Logout 注销当前令牌。
func (s *Service) Logout(ctx context.Context, token string) error {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil
	}
	return s.sessions.DeleteByHash(ctx, hashToken(token))
}

// Authenticate 依据原始令牌解析当前操作者, 令牌无效、过期或用户停用时返回 false。
func (s *Service) Authenticate(ctx context.Context, token string) (*Principal, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, nil
	}
	session, err := s.sessions.GetByHash(ctx, hashToken(token))
	if err != nil {
		return nil, err
	}
	if session == nil || time.Now().After(session.ExpiresAt) {
		return nil, nil
	}
	user, err := s.users.GetByID(ctx, session.UserID)
	if err != nil {
		return nil, err
	}
	if !user.Active {
		return nil, nil
	}
	return user.AsPrincipal(), nil
}

// Profile 是返回给前端的用户信息(不含口令)。
type Profile struct {
	ID          uint     `json:"id"`
	Username    string   `json:"username"`
	DisplayName string   `json:"display_name"`
	Role        string   `json:"role"`
	RoleLabel   string   `json:"role_label"`
	Team        string   `json:"team"`
	Active      bool     `json:"active"`
	Permissions []string `json:"permissions"`
}

// Profile 组装对外用户档案。
func (u *User) Profile() *Profile {
	return &Profile{
		ID:          u.ID,
		Username:    u.Username,
		DisplayName: u.DisplayName,
		Role:        u.Role,
		RoleLabel:   RoleLabel(u.Role),
		Team:        u.Team,
		Active:      u.Active,
		Permissions: PermissionsOf(u.Role),
	}
}

// AsPrincipal 将用户转换为请求上下文操作者。
func (u *User) AsPrincipal() *Principal {
	return &Principal{
		ID:          u.ID,
		Username:    u.Username,
		DisplayName: u.DisplayName,
		Role:        u.Role,
		Team:        u.Team,
	}
}

// CurrentProfile 返回当前操作者档案。
func (s *Service) CurrentProfile(principal *Principal) *Profile {
	if principal == nil {
		return nil
	}
	return &Profile{
		ID:          principal.ID,
		Username:    principal.Username,
		DisplayName: principal.DisplayName,
		Role:        principal.Role,
		RoleLabel:   RoleLabel(principal.Role),
		Team:        principal.Team,
		Active:      true,
		Permissions: principal.Permissions(),
	}
}

// WriteAudit 追加一条审计记录。供审计中间件在请求结束时统一调用。
func (s *Service) WriteAudit(ctx context.Context, entry *AuditLog) {
	if entry == nil {
		return
	}
	if err := s.audits.Create(ctx, entry); err != nil {
		// 审计失败只记录日志, 不阻断主流程; 但拒绝类记录失败要显著告警。
		reportAuditFailure(entry, err)
	}
}

// ListAudit 分页查询审计日志。
func (s *Service) ListAudit(ctx context.Context, filter AuditFilter, page, pageSize int) ([]AuditLog, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 20
	}
	return s.audits.List(ctx, filter, (page-1)*pageSize, pageSize)
}

// NewUser 构造用户(含口令哈希), 供管理岗新增与系统种子数据复用, 保证口令哈希口径一致。
func NewUser(username, password, displayName, role, team string) (*User, error) {
	if !IsValidRole(role) {
		return nil, apperr.BadRequest("非法的角色: %s", role)
	}
	hashed, err := hashPassword(password)
	if err != nil {
		return nil, apperr.Internal("加密口令失败").WithCause(err)
	}
	return &User{
		Username:     strings.TrimSpace(username),
		PasswordHash: hashed,
		DisplayName:  strings.TrimSpace(displayName),
		Role:         role,
		Team:         strings.TrimSpace(team),
		Active:       true,
	}, nil
}

// CreateUser 管理岗新增用户。
func (s *Service) CreateUser(ctx context.Context, req UserUpsertRequest) (*User, error) {
	username := strings.TrimSpace(req.Username)
	if username == "" {
		return nil, apperr.BadRequest("用户名不能为空")
	}
	displayName := strings.TrimSpace(req.DisplayName)
	if displayName == "" {
		return nil, apperr.BadRequest("姓名不能为空")
	}
	if !IsValidRole(req.Role) {
		return nil, apperr.BadRequest("非法的角色: %s", req.Role)
	}
	password := strings.TrimSpace(req.Password)
	if password == "" {
		return nil, apperr.BadRequest("初始密码不能为空")
	}

	exists, err := s.users.ExistsByUsername(ctx, username, 0)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, apperr.Conflict("用户名已存在: %s", username)
	}

	hashed, err := hashPassword(password)
	if err != nil {
		return nil, apperr.Internal("加密口令失败").WithCause(err)
	}
	user := &User{
		Username:     username,
		PasswordHash: hashed,
		DisplayName:  displayName,
		Role:         req.Role,
		Team:         strings.TrimSpace(req.Team),
		Active:       true,
	}
	if err := s.users.Create(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}

// UserUpsertRequest 新增/修改用户请求。
type UserUpsertRequest struct {
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Role        string `json:"role"`
	Team        string `json:"team"`
	Password    string `json:"password"`
	Active      *bool  `json:"active"`
}

// ListUsers 分页查询用户。
func (s *Service) ListUsers(ctx context.Context, filter UserFilter, page, pageSize int) ([]User, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 20
	}
	return s.users.List(ctx, filter, (page-1)*pageSize, pageSize)
}

// UpdateUser 修改用户信息/角色/状态。
// 返回 (变更摘要, 是否影响登录状态), 供上层写审计与强制下线。
func (s *Service) UpdateUser(ctx context.Context, id uint, req UserUpsertRequest) (*User, []string, bool, error) {
	user, err := s.users.GetByID(ctx, id)
	if err != nil {
		return nil, nil, false, err
	}

	changes := make([]string, 0, 4)
	logoutRequired := false

	if req.DisplayName != "" {
		value := strings.TrimSpace(req.DisplayName)
		if value != user.DisplayName {
			changes = append(changes, "姓名:"+user.DisplayName+"→"+value)
			user.DisplayName = value
		}
	}
	if req.Team != "" {
		value := strings.TrimSpace(req.Team)
		if value != user.Team {
			changes = append(changes, "班组:"+user.Team+"→"+value)
			user.Team = value
		}
	}
	if req.Role != "" && req.Role != user.Role {
		if !IsValidRole(req.Role) {
			return nil, nil, false, apperr.BadRequest("非法的角色: %s", req.Role)
		}
		changes = append(changes, "角色:"+RoleLabel(user.Role)+"→"+RoleLabel(req.Role))
		user.Role = req.Role
		logoutRequired = true
	}
	if req.Password != "" {
		hashed, err := hashPassword(strings.TrimSpace(req.Password))
		if err != nil {
			return nil, nil, false, apperr.Internal("加密口令失败").WithCause(err)
		}
		user.PasswordHash = hashed
		changes = append(changes, "已重置密码")
		logoutRequired = true
	}
	if req.Active != nil && *req.Active != user.Active {
		if !*req.Active {
			// 不允许停用最后一个在职管理员, 避免系统失去管理入口。
			adminCount, err := s.users.CountAdmins(ctx, true)
			if err != nil {
				return nil, nil, false, err
			}
			if user.Role == RoleManager && adminCount <= 1 {
				return nil, nil, false, apperr.Conflict("至少保留一个启用状态的管理岗账号")
			}
		}
		user.Active = *req.Active
		if *req.Active {
			changes = append(changes, "状态:停用→启用")
		} else {
			changes = append(changes, "状态:启用→停用")
		}
		logoutRequired = true
	}

	if err := s.users.Save(ctx, user); err != nil {
		return nil, nil, false, err
	}
	if logoutRequired {
		if err := s.sessions.DeleteByUser(ctx, user.ID); err != nil {
			reportAuditFailure(nil, err)
		}
	}
	return user, changes, logoutRequired, nil
}

// GetUser 查询单个用户。
func (s *Service) GetUser(ctx context.Context, id uint) (*User, error) {
	return s.users.GetByID(ctx, id)
}

// reportAuditFailure 记录审计写入失败, 绝不阻断或改写主业务结果。
func reportAuditFailure(entry *AuditLog, err error) {
	attrs := make([]any, 0, 4)
	attrs = append(attrs, "error", err)
	if entry != nil {
		attrs = append(attrs, "audit_action", entry.Action, "audit_result", entry.Result)
	}
	slog.Error("写入审计记录失败", attrs...)
}
