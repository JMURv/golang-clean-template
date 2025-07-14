package config

import "time"

type ctxKey string

const (
	UidKey ctxKey = "uid"
	IpKey  ctxKey = "ip"
	UaKey  ctxKey = "ua"
)

const (
	DefaultPage      = 1
	DefaultSize      = 40
	DefaultCacheTime = time.Hour
	MinCacheTime     = time.Minute * 5
	MaxMemory        = 10 << 20 // 10 MB
)

const (
	AccessCookieName     = "access"
	RefreshCookieName    = "refresh"
	AccessTokenDuration  = time.Minute * 30
	RefreshTokenDuration = time.Hour * 24 * 7
)
