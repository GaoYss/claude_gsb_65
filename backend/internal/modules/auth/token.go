package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// tokenPayload 是登录令牌的载荷, 只携带身份, 角色以数据库实时查询为准,
// 因此调整用户角色/停用账号后立即生效, 无需等待令牌过期。
type tokenPayload struct {
	UserID    uint   `json:"uid"`
	Username  string `json:"usr"`
	IssuedAt  int64  `json:"iat"`
	ExpiresAt int64  `json:"exp"`
}

// TokenService 负责签发与校验 HMAC-SHA256 签名的登录令牌(无状态, 不依赖额外存储)。
type TokenService struct {
	secret []byte
	ttl    time.Duration
}

// NewTokenService 构造令牌服务。
func NewTokenService(secret string, ttl time.Duration) *TokenService {
	if ttl <= 0 {
		ttl = 12 * time.Hour
	}
	return &TokenService{secret: []byte(secret), ttl: ttl}
}

// Issue 为指定用户签发令牌。
func (s *TokenService) Issue(userID uint, username string, now time.Time) (string, time.Time, error) {
	expiresAt := now.Add(s.ttl)
	payload := tokenPayload{
		UserID:    userID,
		Username:  username,
		IssuedAt:  now.Unix(),
		ExpiresAt: expiresAt.Unix(),
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("生成登录令牌失败: %w", err)
	}
	body := base64.RawURLEncoding.EncodeToString(raw)
	return body + "." + s.sign(body), expiresAt, nil
}

// Parse 校验签名与有效期, 返回令牌载荷。
func (s *TokenService) Parse(token string, now time.Time) (*tokenPayload, error) {
	parts := strings.SplitN(strings.TrimSpace(token), ".", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return nil, errors.New("令牌格式不正确")
	}
	if !hmac.Equal([]byte(parts[1]), []byte(s.sign(parts[0]))) {
		return nil, errors.New("令牌签名无效")
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, errors.New("令牌编码无效")
	}
	var payload tokenPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, errors.New("令牌内容无效")
	}
	if payload.UserID == 0 {
		return nil, errors.New("令牌缺少用户身份")
	}
	if now.Unix() >= payload.ExpiresAt {
		return nil, errors.New("登录已过期, 请重新登录")
	}
	return &payload, nil
}

func (s *TokenService) sign(body string) string {
	mac := hmac.New(sha256.New, s.secret)
	mac.Write([]byte(body))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

// BearerToken 从 Authorization 头中取出令牌。
func BearerToken(header string) string {
	const prefix = "Bearer "
	if len(header) > len(prefix) && strings.EqualFold(header[:len(prefix)], prefix) {
		return strings.TrimSpace(header[len(prefix):])
	}
	return strings.TrimSpace(header)
}

// TTL 返回令牌有效期。
func (s *TokenService) TTL() time.Duration { return s.ttl }
