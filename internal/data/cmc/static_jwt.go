// static_jwt.go —— 静态 JWTProvider：中继径下 o-core 零会话件（CMC 个人盘
// 单会话制——任何 Go 侧 createsession 都会踢掉浏览器 web 会话,SG 2026-11-16
// 乒乓实证;中继不验 auth 帧,REST 由浏览器页内 fetch 代发持自己会话）。
package cmc

import "context"

// StaticJWT 常量令牌——中继 ack 全收,令牌内容无所谓;Login=空操作(无会话可建)。
type StaticJWT string

// NewStaticJWT 中继径零会话 auth 件。
func NewStaticJWT(s string) StaticJWT { return StaticJWT(s) }

// JWT 返回常量串。
func (s StaticJWT) JWT(context.Context) (string, error) { return string(s), nil }

// Login 空操作——中继径不建 Go 会话(建即踢浏览器)。
func (s StaticJWT) Login(context.Context) error { return nil }
