package auth

import (
	"context"
	"strings"

	"github.com/gin-gonic/gin"

	"streetlight/internal/apperr"
	"streetlight/internal/middleware"
	"streetlight/internal/response"
)

// gin 上下文与标准上下文使用的键。
const (
	ContextPrincipal   = "auth_principal"    // gin.Context 中的当前操作者
	ContextAction      = "auth_action"       // 本路由要求的权限点, 也是审计动作名
	ContextAuditDetail = "auth_audit_detail" // handler 补充的审计明细(如改派前后负责人)
	ContextAuditNo     = "auth_audit_no"     // handler 补充的资源业务单号
)

type principalCtxKey struct{}

// WithPrincipal 将操作者写入标准 context, 供不依赖 gin 的 service 层读取。
func WithPrincipal(ctx context.Context, principal *Principal) context.Context {
	if principal == nil {
		return ctx
	}
	return context.WithValue(ctx, principalCtxKey{}, principal)
}

// PrincipalFromContext 从标准 context 读取当前操作者, 未登录返回 nil。
func PrincipalFromContext(ctx context.Context) *Principal {
	if value, ok := ctx.Value(principalCtxKey{}).(*Principal); ok {
		return value
	}
	return nil
}

// PrincipalFrom 从 gin.Context 读取当前操作者。
func PrincipalFrom(c *gin.Context) *Principal {
	if value, ok := c.Get(ContextPrincipal); ok {
		if principal, ok := value.(*Principal); ok {
			return principal
		}
	}
	return nil
}

// bearerToken 从 Authorization 头提取 Bearer 令牌。
func bearerToken(c *gin.Context) string {
	header := strings.TrimSpace(c.GetHeader("Authorization"))
	if header == "" {
		return ""
	}
	parts := strings.SplitN(header, " ", 2)
	if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
		return strings.TrimSpace(parts[1])
	}
	return strings.TrimSpace(header)
}

// Authenticate 解析登录令牌并注入当前操作者; 令牌缺失时不拦截, 由权限中间件决定是否放行。
func Authenticate(service *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := bearerToken(c)
		if token != "" {
			principal, err := service.Authenticate(c.Request.Context(), token)
			if err != nil {
				// 令牌解析异常按未认证处理, 具体放行与否交给后续中间件。
				principal = nil
			}
			if principal != nil {
				c.Set(ContextPrincipal, principal)
				c.Request = c.Request.WithContext(WithPrincipal(c.Request.Context(), principal))
			}
		}
		c.Next()
	}
}

// RequireAuth 仅要求已登录, 不限制具体权限点, 用于 /auth/me、/auth/logout 等所有角色可用的接口。
func RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		if PrincipalFrom(c) == nil {
			response.Fail(c, apperr.Unauthorized("未登录或登录状态已失效, 请重新登录"))
			c.Abort()
			return
		}
		c.Next()
	}
}

// RequirePermission 要求当前操作者拥有指定权限点, 否则拒绝并明确指出缺少的授权项。
// 拒绝由审计中间件统一落库; 这里只负责拦截与响应。
func RequirePermission(permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(ContextAction, permission)
		principal := PrincipalFrom(c)
		if principal == nil {
			response.Fail(c, apperr.Unauthorized("未登录或登录状态已失效, 请重新登录"))
			c.Abort()
			return
		}
		if !principal.Can(permission) {
			denied := DeniedError{Permission: permission}
			response.Fail(c, apperr.Forbidden(permission, denied.Error()))
			c.Abort()
			return
		}
		c.Next()
	}
}

// Forbidden 构造一个带缺失权限项的 403 业务错误, 供 service 层归属校验失败时使用。
// reason 用于说明数据范围限制(例如 "仅限本人负责的维修记录")。
func Forbidden(permission, reason string) error {
	return apperr.Forbidden(permission, DeniedError{Permission: permission, Reason: reason}.Error())
}

// requestID 读取链路 ID, 便于审计记录与访问日志关联。
func requestID(c *gin.Context) string {
	return middleware.RequestIDFrom(c)
}
