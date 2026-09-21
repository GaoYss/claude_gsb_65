package auth

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"streetlight/internal/apperr"
	"streetlight/internal/response"
)

// ginContextKey 当前用户在 gin.Context 中的键。
const ginContextKey = "current_user"

// Guard 是全系统统一的鉴权守卫: 登录态解析、权限判定、越权拒绝与审计留痕
// 都收敛在这里, 保证页面、接口、导出、看板使用同一个判定结论。
type Guard struct {
	users  *UserRepository
	tokens *TokenService
	audit  *AuditService
}

// NewGuard 构造鉴权守卫。
func NewGuard(users *UserRepository, tokens *TokenService, audit *AuditService) *Guard {
	return &Guard{users: users, tokens: tokens, audit: audit}
}

// Audit 暴露审计服务, 供需要显式记录的处理器(如导出、登录)使用。
func (g *Guard) Audit() *AuditService { return g.audit }

// CurrentUser 从 gin 上下文取出当前登录用户, 未登录返回 nil。
func CurrentUser(c *gin.Context) *Principal {
	if value, ok := c.Get(ginContextKey); ok {
		if user, ok := value.(*Principal); ok {
			return user
		}
	}
	return nil
}

// Authenticate 解析登录令牌并实时加载用户。
// 角色与启停状态以数据库实时结果为准, 因此权限变更/停用立即生效。
// required 为 true 时, 缺失或无效令牌直接拒绝并留痕; 为 false 时允许匿名通过。
func (g *Guard) Authenticate(required bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := BearerToken(c.GetHeader("Authorization"))
		if token == "" {
			if required {
				g.denyAnonymous(c, "auth.authenticate", "auth", "缺少登录令牌")
				return
			}
			c.Next()
			return
		}

		payload, err := g.tokens.Parse(token, time.Now())
		if err != nil {
			if required {
				g.denyAnonymous(c, "auth.authenticate", "auth", err.Error())
				return
			}
			c.Next()
			return
		}

		user, err := g.users.GetByID(c.Request.Context(), payload.UserID)
		if err != nil {
			if required {
				g.denyAnonymous(c, "auth.authenticate", "auth", "登录账号不存在或已被删除")
				return
			}
			c.Next()
			return
		}
		if !user.Active {
			g.denyAnonymous(c, "auth.authenticate", "auth", "账号已被停用")
			return
		}

		current := &Principal{
			ID:          user.ID,
			Username:    user.Username,
			DisplayName: user.DisplayName,
			Role:        user.Role,
		}
		c.Set(ginContextKey, current)
		c.Request = c.Request.WithContext(current.WithContext(c.Request.Context()))
		c.Next()
	}
}

// Require 返回要求指定权限点的中间件。这是接口层唯一的权限判定入口:
//   - 未登录: 401 并记录 denied 审计;
//   - 已登录但缺少权限: 403, 响应与审计都明确指出缺失的授权点;
//   - 校验通过且为写操作(非 GET): 在处理器成功(HTTP < 400)后记录 allowed 审计。
func (g *Guard) Require(permission, action, resource string) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := CurrentUser(c)
		if user == nil {
			g.denyAnonymous(c, action, resource, "未登录或登录已失效")
			return
		}
		if !user.Can(permission) {
			g.denyForbidden(c, user, action, resource, resourceID(c), permission)
			return
		}

		c.Next()

		// 查询类(GET)默认不记录成功日志以免淹没关键操作; 写操作成功必留痕。
		if isWriteMethod(c.Request.Method) && c.Writer.Status() < 400 {
			g.audit.Record(c, Entry{
				Actor:       user,
				Action:      action,
				Resource:    resource,
				ResourceID:  resourceID(c),
				Result:      AuditResultAllowed,
				Description: PermissionLabel(permission),
			})
		}
	}
}

// DenyOwner 在处理器内部发生资源级归属越权(如维修人员操作他人负责的记录)时调用,
// 记录 denied 审计并返回指明所需授权(repair.manage)的 403。
func (g *Guard) DenyOwner(c *gin.Context, action, resource, resourceID, reason string) {
	user := CurrentUser(c)
	g.denyForbidden(c, user, action, resource, resourceID, PermRepairManage, reason)
}

// EnsureOwner 在处理器内部做资源级归属判定: 管理岗(有 repair.manage)放行,
// 否则仅记录负责人本人可操作。校验失败直接写拒绝响应, 返回 false 表示已拦截。
func (g *Guard) EnsureOwner(c *gin.Context, assigneeID uint, action, resource, resourceID string) bool {
	user := CurrentUser(c)
	if user != nil && (user.Can(PermRepairManage) || assigneeID == user.ID) {
		return true
	}
	reason := "该记录由其他维修人员负责, 你只能推进自己负责的记录"
	g.denyForbidden(c, user, action, resource, resourceID, PermRepairManage, reason)
	return false
}

// denyAnonymous 处理未认证访问: 记录审计后返回 401。
func (g *Guard) denyAnonymous(c *gin.Context, action, resource, reason string) {
	g.audit.Record(c, Entry{
		Action:             action,
		Resource:           resource,
		ResourceID:         resourceID(c),
		Result:             AuditResultDenied,
		RequiredPermission: "",
		Reason:             reason,
	})
	err := apperr.Unauthenticated("未登录或登录已失效, 请先登录")
	c.AbortWithStatusJSON(err.Status, response.Envelope{Code: err.Code, Message: err.Message})
}

// denyForbidden 处理越权访问: 记录审计并返回 403, 明确指出缺失的授权点。
func (g *Guard) denyForbidden(c *gin.Context, user *Principal, action, resource, resourceID, permission string, reasons ...string) {
	reason := ""
	if len(reasons) > 0 {
		reason = strings.TrimSpace(reasons[0])
	}
	message := "无权执行该操作"
	if label := PermissionLabel(permission); label != "" {
		if user != nil {
			message = "当前角色【" + RoleLabel(user.Role) + "】无权执行「" + actionLabel(action) +
				"」, 缺少授权: " + permission + "(" + label + ")"
		} else {
			message = "该操作需要授权: " + permission + "(" + label + ")"
		}
	}
	if reason != "" {
		message += "。" + reason
	}

	g.audit.Record(c, Entry{
		Actor:              user,
		Action:             action,
		Resource:           resource,
		ResourceID:         resourceID,
		Result:             AuditResultDenied,
		RequiredPermission: permission,
		Reason:             reason,
		Description:        actionLabel(action),
	})

	err := apperr.PermissionDenied(permission, message)
	c.AbortWithStatusJSON(err.Status, response.Envelope{Code: err.Code, Message: err.Message, Details: err.Details})
}

// resourceID 从路径参数中尽力提取资源标识。
func resourceID(c *gin.Context) string {
	for _, key := range []string{"id", "faultId", "lampId", "repairId"} {
		if value := strings.TrimSpace(c.Param(key)); value != "" {
			return value
		}
	}
	return ""
}

func isWriteMethod(method string) bool {
	switch method {
	case "POST", "PUT", "PATCH", "DELETE":
		return true
	default:
		return false
	}
}

// actionLabel 将动作标识转为中文说明, 未配置时回退为原值。
func actionLabel(action string) string {
	if label, ok := actionLabels[action]; ok {
		return label
	}
	return action
}

// actionLabels 审计动作文案。
var actionLabels = map[string]string{
	"lamp.create":     "新增路灯",
	"lamp.update":     "修改路灯",
	"lamp.delete":     "删除路灯",
	"fault.register":  "登记故障",
	"fault.update":    "修改故障",
	"fault.close":     "关闭故障",
	"fault.delete":    "删除故障",
	"fault.reassign":  "改派故障负责人",
	"fault.exception": "故障例外流转",
	"repair.create":   "录入/开工维修",
	"repair.update":   "修改维修记录",
	"repair.finish":   "完工维修",
	"repair.delete":   "删除维修记录",
	"repair.reassign": "改派维修负责人",
	"export.fault":    "导出故障数据",
	"export.repair":   "导出维修数据",
	"auth.login":      "登录",
	"auth.logout":     "登出",
	"user.manage":     "维护用户与权限",
}
