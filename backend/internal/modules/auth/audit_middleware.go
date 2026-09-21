package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"streetlight/internal/middleware"
	"streetlight/internal/response"
)

// Audit 审计中间件: 统一记录每一次写操作与每一次越权拒绝。
// 它是审计落库的唯一入口, 业务 handler/service 不直接写审计, 保证口径一致:
//   - 写操作(POST/PUT/PATCH/DELETE)一律记录, 放行记 allowed, 被拒记 denied;
//   - 任何返回 403 的请求(含读接口越权尝试)都记录 denied, 并写明缺失权限项;
//   - 401(未登录)只在写接口记录, 读接口的匿名访问不记审计以免被探测刷库。
func Audit(service *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		method := c.Request.Method
		statusCode := c.Writer.Status()
		isWrite := method == http.MethodPost || method == http.MethodPut ||
			method == http.MethodPatch || method == http.MethodDelete
		denied := statusCode == http.StatusForbidden

		// 只记录写操作与越权拒绝; 未登录的匿名读请求不记审计, 避免被探测刷库。
		if !isWrite && !denied {
			return
		}
		// 未登录尝试写接口(isWrite + 401)同样向下记录为 denied。

		principal := PrincipalFrom(c)
		action, resourceType := resolveAction(c)

		result := AuditResultAllowed
		missing := ""
		switch {
		case denied:
			// 403: 已认证但越权, 必须记录缺失的授权项。
			result = AuditResultDenied
			missing = c.GetString(response.ContextDeniedPermission)
			if missing == "" {
				missing = action
			}
		case statusCode == http.StatusUnauthorized:
			// 401: 未登录/会话失效/登录失败, 同样视为被拒绝并留痕。
			result = AuditResultDenied
		}

		entry := &AuditLog{
			Action:            action,
			ResourceType:      resourceType,
			ResourceID:        pathID(c),
			Result:            result,
			MissingPermission: missing,
			Method:            method,
			Path:              truncate(c.Request.URL.Path, 255),
			RequestID:         middleware.RequestIDFrom(c),
			ClientIP:          c.ClientIP(),
			StatusCode:        statusCode,
		}
		if principal != nil {
			entry.ActorID = principal.ID
			entry.ActorUsername = principal.Username
			entry.ActorRole = principal.Role
		}
		entry.ResourceNo = c.Param("no")
		if no := strings.TrimSpace(c.GetString(ContextAuditNo)); no != "" {
			entry.ResourceNo = no
		}
		if detail := strings.TrimSpace(c.GetString(ContextAuditDetail)); detail != "" {
			entry.Detail = truncate(detail, 1000)
		}

		// 用独立 context 写入, 避免客户端断开或请求结束导致审计(尤其越权记录)丢失。
		service.WriteAudit(context.WithoutCancel(c.Request.Context()), entry)
	}
}

// resolveAction 解析审计动作名与资源类型: 优先取权限中间件写入的权限点,
// 登录/登出等无权限点的路由按路径归类。
func resolveAction(c *gin.Context) (action, resourceType string) {
	if value := c.GetString(ContextAction); value != "" {
		return value, resourceOf(value)
	}
	path := c.Request.URL.Path
	switch {
	case strings.HasSuffix(path, "/auth/login"):
		return "auth:login", "auth"
	case strings.HasSuffix(path, "/auth/logout"):
		return "auth:logout", "auth"
	default:
		last := path
		if index := strings.LastIndex(last, "/"); index >= 0 {
			last = last[index+1:]
		}
		return methodAction(c.Request.Method, last), "unknown"
	}
}

// resourceOf 从权限点提取资源类型, 例如 "repair:finish" -> "repair"。
func resourceOf(permission string) string {
	if index := strings.Index(permission, ":"); index > 0 {
		return permission[:index]
	}
	return permission
}

// methodAction 为无权限点的写操作生成兜底动作名。
func methodAction(method, tail string) string {
	var verb string
	switch method {
	case http.MethodPost:
		verb = "create"
	case http.MethodPut, http.MethodPatch:
		verb = "update"
	case http.MethodDelete:
		verb = "delete"
	default:
		verb = strings.ToLower(method)
	}
	if tail == "" {
		return verb
	}
	return tail + ":" + verb
}

// pathID 尽力从路径参数中解析资源主键, 失败返回 0。
func pathID(c *gin.Context) uint {
	for _, name := range []string{"id", "faultId", "lampId", "repairId", "userId"} {
		if raw := strings.TrimSpace(c.Param(name)); raw != "" {
			var value uint
			valid := true
			for _, ch := range raw {
				if ch < '0' || ch > '9' {
					valid = false
					break
				}
				value = value*10 + uint(ch-'0')
			}
			if valid && value > 0 {
				return value
			}
		}
	}
	return 0
}

func truncate(value string, max int) string {
	if len(value) <= max {
		return value
	}
	return value[:max]
}
