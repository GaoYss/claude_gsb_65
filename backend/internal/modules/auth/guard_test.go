package auth_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"

	"streetlight/internal/modules/auth"
)

type loginResponse struct {
	Code int `json:"-"`
	Data struct {
		Token       string   `json:"token"`
		Permissions []string `json:"permissions"`
	} `json:"data"`
}

type apiError struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details"`
}

// setupGuardApp 构造带完整登录与鉴权链路的测试引擎。
func setupGuardApp(t *testing.T) (*gin.Engine, *gorm.DB, *auth.Module) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	db, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{
		Logger:         logger.Default.LogMode(logger.Silent),
		NamingStrategy: schema.NamingStrategy{SingularTable: true},
	})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, db.AutoMigrate(&auth.User{}, &auth.AuditLog{}))
	require.NoError(t, auth.InstallAuditGuard(db))

	module := auth.New(db, "test-secret", time.Hour)
	require.NoError(t, module.Service().EnsureSeedUsers(context.Background()))

	engine := gin.New()
	api := engine.Group("/api/v1")
	module.RegisterRoutes(api)

	// 挂三个不同权限点的探针端点, 模拟业务写操作与导出。
	probe := api.Group("/probe")
	probe.GET("/view", module.Guard().Authenticate(true), module.Guard().Require(auth.PermStatusView, "probe.view", "probe"), func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})
	probe.POST("/register", module.Guard().Authenticate(true), module.Guard().Require(auth.PermFaultRegister, "probe.register", "probe"), func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})
	probe.POST("/reassign", module.Guard().Authenticate(true), module.Guard().Require(auth.PermRepairReassign, "probe.reassign", "probe"), func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})

	return engine, db, module
}

func login(t *testing.T, engine *gin.Engine, username, password string) string {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"username": username, "password": password})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, "登录应成功: %s", w.Body.String())

	var resp struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.NotEmpty(t, resp.Data.Token)
	return resp.Data.Token
}

func doRequest(t *testing.T, engine *gin.Engine, method, path, token string) (int, apiError) {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	var envelope apiError
	if w.Code >= 400 {
		_ = json.Unmarshal(w.Body.Bytes(), &envelope)
	}
	return w.Code, envelope
}

// TestCrossRoleDenial 验证三类角色的操作范围分离, 越权返回 403 并指明缺失授权。
func TestCrossRoleDenial(t *testing.T) {
	engine, _, _ := setupGuardApp(t)

	registrar := login(t, engine, "registrar", "registrar123")
	repairman := login(t, engine, "repairman", "repair123")
	admin := login(t, engine, "admin", "admin123")

	// 登记人员: 可查询与登记, 不能改派。
	status, _ := doRequest(t, engine, http.MethodGet, "/api/v1/probe/view", registrar)
	require.Equal(t, http.StatusOK, status)
	status, _ = doRequest(t, engine, http.MethodPost, "/api/v1/probe/register", registrar)
	require.Equal(t, http.StatusOK, status)
	status, errBody := doRequest(t, engine, http.MethodPost, "/api/v1/probe/reassign", registrar)
	require.Equal(t, http.StatusForbidden, status)
	require.Equal(t, "PERMISSION_DENIED", errBody.Code)
	require.Equal(t, auth.PermRepairReassign, errBody.Details["required_permission"], "拒绝必须指明缺失的授权点")
	require.Contains(t, errBody.Message, auth.PermRepairReassign)

	// 维修人员: 可查询, 不能登记故障, 也不能改派。
	status, _ = doRequest(t, engine, http.MethodGet, "/api/v1/probe/view", repairman)
	require.Equal(t, http.StatusOK, status)
	status, errBody = doRequest(t, engine, http.MethodPost, "/api/v1/probe/register", repairman)
	require.Equal(t, http.StatusForbidden, status)
	require.Equal(t, auth.PermFaultRegister, errBody.Details["required_permission"])
	status, errBody = doRequest(t, engine, http.MethodPost, "/api/v1/probe/reassign", repairman)
	require.Equal(t, http.StatusForbidden, status)
	require.Equal(t, auth.PermRepairReassign, errBody.Details["required_permission"])

	// 管理岗: 全部放行。
	for _, call := range []struct {
		method, path string
	}{
		{http.MethodGet, "/api/v1/probe/view"},
		{http.MethodPost, "/api/v1/probe/register"},
		{http.MethodPost, "/api/v1/probe/reassign"},
	} {
		status, _ := doRequest(t, engine, call.method, call.path, admin)
		require.Equal(t, http.StatusOK, status, "管理岗应可访问 %s", call.path)
	}
}

// TestUnauthenticatedAndBadToken 验证未登录与非法令牌被拒并留痕。
func TestUnauthenticatedAndBadToken(t *testing.T) {
	engine, _, _ := setupGuardApp(t)

	status, errBody := doRequest(t, engine, http.MethodGet, "/api/v1/probe/view", "")
	require.Equal(t, http.StatusUnauthorized, status)
	require.Equal(t, "UNAUTHENTICATED", errBody.Code)

	status, _ = doRequest(t, engine, http.MethodGet, "/api/v1/probe/view", "forged.token.value")
	require.Equal(t, http.StatusUnauthorized, status)
}

// TestDeniedOperationsAreAudited 验证越权操作与登录失败都留下不可改写的审计记录。
func TestDeniedOperationsAreAudited(t *testing.T) {
	engine, db, _ := setupGuardApp(t)
	repairman := login(t, engine, "repairman", "repair123")

	// 维修人员尝试登记故障 -> 拒绝。
	doRequest(t, engine, http.MethodPost, "/api/v1/probe/register", repairman)
	// 错误密码登录 -> 拒绝。
	body, _ := json.Marshal(map[string]string{"username": "repairman", "password": "wrong-pass"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)

	var deniedCount int64
	require.NoError(t, db.Model(&auth.AuditLog{}).
		Where("result = ?", auth.AuditResultDenied).Count(&deniedCount).Error)
	require.GreaterOrEqual(t, deniedCount, int64(2), "越权与登录失败都应留痕")

	var reassignDenied int64
	require.NoError(t, db.Model(&auth.AuditLog{}).
		Where("action = ? AND required_permission = ? AND actor_username = ?",
			"probe.register", auth.PermFaultRegister, "repairman").
		Count(&reassignDenied).Error)
	require.Equal(t, int64(1), reassignDenied, "审计需记录操作者、动作与缺失授权")
}
