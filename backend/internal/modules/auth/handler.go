package auth

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"streetlight/internal/httpx"
	"streetlight/internal/response"
)

// Handler 处理认证、用户管理与审计查询请求。
type Handler struct {
	service *Service
	guard   *Guard
}

// NewHandler 构造认证处理器。
func NewHandler(service *Service, guard *Guard) *Handler {
	return &Handler{service: service, guard: guard}
}

// Login 登录。
func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := httpx.BindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	result, err := h.service.Login(c.Request.Context(), req.Username, req.Password, c.ClientIP())
	if err != nil {
		// 登录失败同样留痕(未认证操作者)。
		h.guard.Audit().Record(c, Entry{
			Action: "auth.login", Resource: "auth", ResourceID: req.Username,
			Result: AuditResultDenied, Reason: err.Error(),
		})
		response.Fail(c, err)
		return
	}
	h.guard.Audit().Record(c, Entry{
		Actor: &Principal{
			ID: result.User.ID, Username: result.User.Username,
			DisplayName: result.User.DisplayName, Role: result.User.Role,
		},
		Action: "auth.login", Resource: "auth", ResourceID: result.User.Username,
		Result: AuditResultAllowed, Description: "用户登录",
	})
	response.OK(c, result)
}

// Logout 登出(令牌无状态, 服务端仅记录登出行为)。
func (h *Handler) Logout(c *gin.Context) {
	if user := CurrentUser(c); user != nil {
		h.guard.Audit().Record(c, Entry{
			Actor: user, Action: "auth.logout", Resource: "auth",
			ResourceID: user.Username, Result: AuditResultAllowed, Description: "用户登出",
		})
	}
	response.OK(c, gin.H{"message": "已登出"})
}

// Profile 返回当前登录用户信息与权限快照。
func (h *Handler) Profile(c *gin.Context) {
	result, err := h.service.Profile(c.Request.Context(), CurrentUser(c))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, result)
}

// Catalog 返回角色与权限点目录。
func (h *Handler) Catalog(c *gin.Context) {
	response.OK(c, PermissionCatalogResponse{
		Roles:       RoleCatalog(),
		Permissions: PermissionCatalog(),
	})
}

// ChangePassword 用户修改自身密码。
func (h *Handler) ChangePassword(c *gin.Context) {
	var req ChangePasswordRequest
	if err := httpx.BindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.service.ChangePassword(c.Request.Context(), CurrentUser(c), req.OldPassword, req.NewPassword); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, gin.H{"message": "密码已修改"})
}

// ListUsers 用户列表。
func (h *Handler) ListUsers(c *gin.Context) {
	keyword := c.Query("keyword")
	role := c.Query("role")
	users, total, err := h.service.ListUsers(c.Request.Context(), keyword, role)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, gin.H{"items": users, "total": total})
}

// CreateUser 新建用户。
func (h *Handler) CreateUser(c *gin.Context) {
	var req UserUpsertRequest
	if err := httpx.BindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	user, err := h.service.CreateUser(c.Request.Context(), req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Created(c, user)
}

// UpdateUser 修改用户角色/显示名/启停。
func (h *Handler) UpdateUser(c *gin.Context) {
	id, err := httpx.ParseID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req UserUpdateRequest
	if err := httpx.BindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	user, err := h.service.UpdateUser(c.Request.Context(), id, req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, user)
}

// ResetPassword 管理岗重置用户密码。
func (h *Handler) ResetPassword(c *gin.Context) {
	id, err := httpx.ParseID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req PasswordResetRequest
	if err := httpx.BindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.service.ResetPassword(c.Request.Context(), id, req.Password); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, gin.H{"message": "密码已重置"})
}

// ListAudit 查询审计日志(只读, 无任何修改入口)。
func (h *Handler) ListAudit(c *gin.Context) {
	var query AuditQuery
	if err := httpx.BindQuery(c, &query); err != nil {
		response.Fail(c, err)
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 20
	}

	entries, total, err := h.guard.Audit().List(c.Request.Context(), AuditFilter{
		Actor: query.Actor, Action: query.Action, Resource: query.Resource,
		Result: query.Result, Keyword: query.Keyword, From: query.From, To: query.To,
	}, (page-1)*pageSize, pageSize)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, response.NewPageData(entries, total, page, pageSize))
}
