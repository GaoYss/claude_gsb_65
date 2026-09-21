package response

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"streetlight/internal/apperr"
)

// CodeOK 表示业务处理成功。
const CodeOK = "OK"

// ContextDeniedPermission 是拒绝(403)时写入 gin.Context 的缺失权限项键名,
// 供审计中间件在请求结束后统一读取, 避免横切逻辑与业务包形成循环依赖。
const ContextDeniedPermission = "denied_permission"

// ContextDeniedReason 是拒绝(403)时写入的补充原因键名。
const ContextDeniedReason = "denied_reason"

// Envelope 是所有接口统一的响应结构。
type Envelope struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
	// MissingPermission 仅 403 时出现, 指明缺少的授权点, 与审计记录一致。
	MissingPermission string `json:"missing_permission,omitempty"`
}

// PageData 是列表接口统一的分页结构。
type PageData[T any] struct {
	Items      []T   `json:"items"`
	Total      int64 `json:"total"`
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	TotalPages int   `json:"total_pages"`
}

// NewPageData 组装分页响应, 保证空列表序列化为 [] 而不是 null。
func NewPageData[T any](items []T, total int64, page, pageSize int) PageData[T] {
	if items == nil {
		items = make([]T, 0)
	}
	totalPages := 0
	if pageSize > 0 {
		totalPages = int((total + int64(pageSize) - 1) / int64(pageSize))
	}
	return PageData[T]{
		Items:      items,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}
}

// OK 返回 200 成功响应。
func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, Envelope{Code: CodeOK, Message: "success", Data: data})
}

// Created 返回 201 创建成功响应。
func Created(c *gin.Context, data any) {
	c.JSON(http.StatusCreated, Envelope{Code: CodeOK, Message: "created", Data: data})
}

// NoContent 返回 204 空响应。
func NoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

// Fail 将任意 error 转换为统一错误响应: 业务错误按自身状态码返回, 其余按 500 处理。
func Fail(c *gin.Context, err error) {
	if appErr, ok := apperr.As(err); ok {
		if appErr.Status >= http.StatusInternalServerError {
			_ = c.Error(err)
		}
		if appErr.Status == http.StatusForbidden {
			// 透传给审计中间件: 被拒操作缺少了哪一项授权。
			c.Set(ContextDeniedPermission, appErr.MissingPermission)
		}
		c.JSON(appErr.Status, Envelope{
			Code:              appErr.Code,
			Message:           appErr.Message,
			MissingPermission: appErr.MissingPermission,
		})
		return
	}

	var ginErr *gin.Error
	if errors.As(err, &ginErr) && ginErr.Type == gin.ErrorTypeBind {
		c.JSON(http.StatusBadRequest, Envelope{Code: apperr.CodeInvalidArgument, Message: "请求参数格式不正确"})
		return
	}

	slog.Error("未处理的服务端错误", "error", err, "path", c.FullPath(), "method", c.Request.Method)
	c.JSON(http.StatusInternalServerError, Envelope{Code: apperr.CodeInternalError, Message: "服务器内部错误, 请稍后重试"})
}
