package auth

import (
	"context"
)

// ctxKey 是当前登录用户在 gin.Context / context.Context 中的键类型。
type ctxKey struct{}

// Principal 是解析登录态后挂在请求上下文中的操作者信息。
type Principal struct {
	ID          uint
	Username    string
	DisplayName string
	Role        string
}

// WithContext 返回携带当前用户的 context。
func (u *Principal) WithContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, ctxKey{}, u)
}

// FromContext 从 context 中取出当前登录用户, 未登录返回 nil。
func FromContext(ctx context.Context) *Principal {
	if ctx == nil {
		return nil
	}
	if user, ok := ctx.Value(ctxKey{}).(*Principal); ok {
		return user
	}
	return nil
}

// Can 判断当前用户是否拥有指定权限点。空用户(未登录)一律无权限。
func (u *Principal) Can(permission string) bool {
	if u == nil {
		return false
	}
	return HasPermission(u.Role, permission)
}

// IsAdmin 判断是否管理岗。
func (u *Principal) IsAdmin() bool {
	return u != nil && u.Role == RoleAdmin
}
