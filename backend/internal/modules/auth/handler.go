package auth

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"streetlight/internal/apperr"
	"streetlight/internal/httpx"
	"streetlight/internal/response"
)

// Handler 处理认证、用户管理与审计查询请求。
type Handler struct {
	service *Service
}

// NewHandler 构造认证处理器。
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Login 登录并签发令牌。
func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := httpx.BindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	result, err := h.service.Login(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		response.Fail(c, err)
		return
	}
	if result.User != nil {
		c.Set(ContextAuditNo, result.User.Username)
	}
	response.OK(c, result)
}

// Logout 注销当前令牌。
func (h *Handler) Logout(c *gin.Context) {
	token := bearerToken(c)
	_ = h.service.Logout(c.Request.Context(), token)
	response.OK(c, gin.H{"success": true})
}

// Me 返回当前登录用户档案与权限点, 前端据此渲染菜单与按钮。
func (h *Handler) Me(c *gin.Context) {
	principal := PrincipalFrom(c)
	if principal == nil {
		response.Fail(c, apperr.Unauthorized("未登录或登录状态已失效"))
		return
	}
	roles := make([]RoleOption, 0, len(Roles()))
	for _, role := range Roles() {
		roles = append(roles, RoleOption{Value: role, Label: RoleLabel(role), Permissions: PermissionsOf(role)})
	}
	response.OK(c, MeResponse{Profile: h.service.CurrentProfile(principal), Roles: roles})
}

// ListUsers 用户列表(管理岗)。
func (h *Handler) ListUsers(c *gin.Context) {
	var query UserListQuery
	if err := httpx.BindQuery(c, &query); err != nil {
		response.Fail(c, err)
		return
	}
	users, total, err := h.service.ListUsers(c.Request.Context(), UserFilter{
		Keyword: query.Keyword,
		Role:    strings.TrimSpace(query.Role),
		Active:  query.Active,
	}, query.Page, query.PageSize)
	if err != nil {
		response.Fail(c, err)
		return
	}
	items := make([]*Profile, 0, len(users))
	for index := range users {
		items = append(items, users[index].Profile())
	}
	response.OK(c, response.NewPageData(items, total, normalizePage(query.Page), normalizePageSize(query.PageSize)))
}

// CreateUser 新增用户(管理岗)。
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
	c.Set(ContextAuditNo, user.Username)
	c.Set(ContextAuditDetail, "新增用户 角色="+RoleLabel(user.Role)+" 班组="+user.Team)
	response.Created(c, user.Profile())
}

// UpdateUser 修改用户信息/角色/状态(管理岗), 角色变更强制下线并写审计明细。
func (h *Handler) UpdateUser(c *gin.Context) {
	id, err := httpx.ParseID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req UserUpsertRequest
	if err := httpx.BindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	user, changes, _, err := h.service.UpdateUser(c.Request.Context(), id, req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	c.Set(ContextAuditNo, user.Username)
	if len(changes) > 0 {
		c.Set(ContextAuditDetail, strings.Join(changes, "; "))
	}
	response.OK(c, user.Profile())
}

// ListAudit 审计日志查询(管理岗)。
func (h *Handler) ListAudit(c *gin.Context) {
	var query AuditListQuery
	if err := httpx.BindQuery(c, &query); err != nil {
		response.Fail(c, err)
		return
	}
	filter := AuditFilter{
		ActorUsername: query.ActorUsername,
		Action:        strings.TrimSpace(query.Action),
		Result:        strings.TrimSpace(query.Result),
		ResourceType:  strings.TrimSpace(query.ResourceType),
		Keyword:       query.Keyword,
	}
	if value := strings.TrimSpace(query.StartDate); value != "" {
		if from, err := time.ParseInLocation("2006-01-02", value, time.Local); err == nil {
			filter.From = &from
		}
	}
	if value := strings.TrimSpace(query.EndDate); value != "" {
		if to, err := time.ParseInLocation("2006-01-02", value, time.Local); err == nil {
			end := to.AddDate(0, 0, 1)
			filter.To = &end
		}
	}
	items, total, err := h.service.ListAudit(c.Request.Context(), filter, query.Page, query.PageSize)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, response.NewPageData(items, total, normalizePage(query.Page), normalizePageSize(query.PageSize)))
}

// PermissionMeta 返回全部权限点定义, 供审计页筛选与前端展示。
func (h *Handler) PermissionMeta(c *gin.Context) {
	type item struct {
		Value string `json:"value"`
		Label string `json:"label"`
	}
	items := make([]item, 0, len(PermissionInfo))
	for value, label := range PermissionInfo {
		items = append(items, item{Value: value, Label: label})
	}
	response.OK(c, gin.H{"permissions": items, "results": []string{AuditResultAllowed, AuditResultDenied}})
}

func normalizePage(page int) int {
	if page < 1 {
		return 1
	}
	return page
}

func normalizePageSize(size int) int {
	if size < 1 {
		return 20
	}
	if size > 200 {
		return 200
	}
	return size
}
