package auth

import "errors"

// 认证与用户管理相关的哨兵错误。
var (
	ErrUserNotFound      = errors.New("用户不存在")
	ErrInvalidCredential = errors.New("用户名或密码不正确")
	ErrUserDisabled      = errors.New("账号已停用, 请联系管理员")
	ErrDuplicateUsername = errors.New("用户名已存在")
)
