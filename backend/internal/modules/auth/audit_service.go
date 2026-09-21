package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"log/slog"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// AuditService 统一记录操作审计。任何鉴权判定(允许或拒绝)都经此落库,
// 保证页面、接口、导出、看板四处的越权操作都留下同一格式的记录。
type AuditService struct {
	repo *AuditRepository
}

// NewAuditService 构造审计服务。
func NewAuditService(repo *AuditRepository) *AuditService {
	return &AuditService{repo: repo}
}

// Entry 是一次待记录操作的结构化描述。
type Entry struct {
	Actor              *Principal
	Action             string // 如 fault.create / export.fault
	Resource           string // fault / repair / lamp / export / auth / user
	ResourceID         string
	Description        string
	Result             string // allowed / denied
	RequiredPermission string // 被拒时缺失的授权点
	Reason             string
}

// Record 从 gin 请求上下文构造并追加一条审计记录。写审计失败只记录日志,
// 不阻断业务(拒绝动作本身已在业务层生效; 但拒绝记录会尽量保留)。
func (s *AuditService) Record(c *gin.Context, entry Entry) {
	s.RecordContext(c.Request.Context(), c, entry)
}

// RecordContext 追加审计记录, httpCtx 用于读取请求快照(可为 nil, 用于非 HTTP 场景测试)。
func (s *AuditService) RecordContext(ctx context.Context, httpCtx *gin.Context, entry Entry) {
	now := time.Now()
	log := &AuditLog{
		OccurredAt:         now,
		CreatedAt:          now,
		Action:             entry.Action,
		Resource:           entry.Resource,
		ResourceID:         trim(entry.ResourceID, 64),
		Description:        trim(entry.Description, 255),
		Result:             entry.Result,
		RequiredPermission: entry.RequiredPermission,
		Reason:             trim(entry.Reason, 255),
		RoleSnapshot:       "-",
		PermSnapshot:       "",
	}

	if entry.Actor != nil {
		log.ActorID = entry.Actor.ID
		log.ActorUsername = entry.Actor.Username
		log.ActorName = entry.Actor.DisplayName
		log.ActorRole = entry.Actor.Role
		log.RoleSnapshot = entry.Actor.Role
		log.PermSnapshot = strings.Join(PermissionsForRole(entry.Actor.Role), ",")
	}
	if log.PermSnapshot == "" {
		log.PermSnapshot = "-"
	}

	if httpCtx != nil {
		log.Method = httpCtx.Request.Method
		log.Path = httpCtx.FullPath()
		if log.Path == "" {
			log.Path = httpCtx.Request.URL.Path
		}
		log.IP = httpCtx.ClientIP()
		log.UserAgent = trim(httpCtx.Request.UserAgent(), 255)
		log.RequestID = requestID(httpCtx)
		log.HTTPStatus = httpCtx.Writer.Status()
		log.RequestHash = fingerprint(httpCtx.Request.Method, log.Path, entry.ResourceID)
	}

	if err := s.repo.Append(ctx, log); err != nil {
		slog.Error("审计日志写入失败", "action", entry.Action, "result", entry.Result, "error", err)
	}
}

// List 查询审计记录。
func (s *AuditService) List(ctx context.Context, filter AuditFilter, offset, limit int) ([]AuditLog, int64, error) {
	return s.repo.List(ctx, filter, offset, limit)
}

// fingerprint 生成 method+path+资源标识的稳定哈希, 便于关联同一次请求的多条记录。
func fingerprint(parts ...string) string {
	joined := strings.Join(parts, "|")
	sum := sha256.Sum256([]byte(joined))
	return hex.EncodeToString(sum[:])[:16]
}

func trim(value string, max int) string {
	value = strings.TrimSpace(value)
	if len(value) > max {
		return value[:max]
	}
	return value
}

func requestID(c *gin.Context) string {
	if value, ok := c.Get("request_id"); ok {
		if id, ok := value.(string); ok {
			return id
		}
	}
	return c.Writer.Header().Get("X-Request-Id")
}
