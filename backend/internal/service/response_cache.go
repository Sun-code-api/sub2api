package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"
)

// ErrResponseCacheMiss 表示响应缓存未命中（或缓存驱动不可用）。
var ErrResponseCacheMiss = errors.New("response cache miss")

const responseCachePrefix = "response_cache:"

// DefaultResponseCacheTTL 是响应缓存默认 TTL。
const DefaultResponseCacheTTL = 60 * time.Second

// DefaultResponseCacheMaxBytes 是允许缓存的响应体默认上限。
const DefaultResponseCacheMaxBytes = 256 * 1024

// responseCachePort 由支持响应缓存的缓存驱动（repository.gatewayCache）实现。
// 独立于 GatewayCache 接口，避免破坏既有测试桩。
type responseCachePort interface {
	GetResponseCache(ctx context.Context, key string) ([]byte, error)
	SetResponseCache(ctx context.Context, key string, payload []byte, ttl time.Duration) error
}

// BuildResponseCacheKey 生成非流式 Chat Completions 响应缓存键：
// sha256(apiKeyID | groupID | model | rawBody)。rawBody 即客户端原始请求体，
// 相同请求（含 messages/system 等）在 TTL 内命中缓存。
func BuildResponseCacheKey(apiKeyID int64, groupID int64, model string, body []byte) string {
	h := sha256.New()
	h.Write([]byte(fmt.Sprintf("%d|%d|%s|", apiKeyID, groupID, model)))
	h.Write(body)
	return responseCachePrefix + hex.EncodeToString(h.Sum(nil))
}

// GetResponseCache 读取响应缓存；未命中或驱动不可用时返回 ErrResponseCacheMiss。
func (s *OpenAIGatewayService) GetResponseCache(ctx context.Context, key string) ([]byte, error) {
	if s == nil || s.cache == nil {
		return nil, ErrResponseCacheMiss
	}
	p, ok := s.cache.(responseCachePort)
	if !ok {
		return nil, ErrResponseCacheMiss
	}
	return p.GetResponseCache(ctx, key)
}

// SetResponseCache 写入响应缓存。驱动不可用或 payload 为空时静默忽略。
func (s *OpenAIGatewayService) SetResponseCache(ctx context.Context, key string, payload []byte, ttl time.Duration) error {
	if s == nil || s.cache == nil || len(payload) == 0 {
		return nil
	}
	p, ok := s.cache.(responseCachePort)
	if !ok {
		return nil
	}
	return p.SetResponseCache(ctx, key, payload, ttl)
}

// ResponseCacheEnabled 报告响应缓存总开关（config gateway.response_cache_enabled）。
func (s *OpenAIGatewayService) ResponseCacheEnabled() bool {
	return s != nil && s.cfg != nil && s.cfg.Gateway.ResponseCacheEnabled
}

// ResponseCacheTTL 返回响应缓存 TTL（config gateway.response_cache_ttl_seconds，默认 60s）。
func (s *OpenAIGatewayService) ResponseCacheTTL() time.Duration {
	if s == nil || s.cfg == nil || s.cfg.Gateway.ResponseCacheTTLSeconds <= 0 {
		return DefaultResponseCacheTTL
	}
	return time.Duration(s.cfg.Gateway.ResponseCacheTTLSeconds) * time.Second
}

// ResponseCacheMaxBytes 返回允许缓存的响应体上限（config
// gateway.response_cache_max_body_bytes，默认 256KB）。
func (s *OpenAIGatewayService) ResponseCacheMaxBytes() int64 {
	if s == nil || s.cfg == nil || s.cfg.Gateway.ResponseCacheMaxBodyBytes <= 0 {
		return DefaultResponseCacheMaxBytes
	}
	return s.cfg.Gateway.ResponseCacheMaxBodyBytes
}
