//go:build unit

package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func plazaTestChannels() []service.AvailableChannel {
	in := 1e-6
	out := 5e-6
	return []service.AvailableChannel{
		{
			ID:     1,
			Name:   "channel-a",
			Status: service.StatusActive,
			Groups: []service.AvailableGroupRef{
				{ID: 10, Name: "public-claude", Platform: "anthropic", RateMultiplier: 1.0, IsExclusive: false},
				{ID: 11, Name: "vip-claude", Platform: "anthropic", RateMultiplier: 0.8, IsExclusive: true},
			},
			SupportedModels: []service.SupportedModel{
				{Name: "claude-sonnet-4-5", Platform: "anthropic", Pricing: &service.ChannelModelPricing{InputPrice: &in, OutputPrice: &out}},
			},
		},
		{
			ID:     2,
			Name:   "channel-b",
			Status: service.StatusActive,
			Groups: []service.AvailableGroupRef{
				{ID: 10, Name: "public-claude", Platform: "anthropic", RateMultiplier: 1.0, IsExclusive: false},
				{ID: 20, Name: "public-openai", Platform: "openai", RateMultiplier: 1.5, IsExclusive: false},
			},
			SupportedModels: []service.SupportedModel{
				{Name: "claude-sonnet-4-5", Platform: "anthropic", Pricing: &service.ChannelModelPricing{InputPrice: &in, OutputPrice: &out}},
				{Name: "gpt-5", Platform: "openai", Pricing: &service.ChannelModelPricing{InputPrice: &in, OutputPrice: &out}},
			},
		},
		{
			ID:     3,
			Name:   "channel-disabled",
			Status: "disabled",
			Groups: []service.AvailableGroupRef{
				{ID: 30, Name: "public-gemini", Platform: "gemini", RateMultiplier: 1.0, IsExclusive: false},
			},
			SupportedModels: []service.SupportedModel{
				{Name: "gemini-2.5-pro", Platform: "gemini"},
			},
		},
	}
}

func TestBuildModelPlaza_MergesChannelsAndDedupesGroups(t *testing.T) {
	allowed := map[int64]struct{}{10: {}, 20: {}}
	entries := buildModelPlaza(plazaTestChannels(), allowed)

	// 停用渠道的 gemini 模型不出现
	require.Len(t, entries, 2)
	// 排序：platform 字母序 anthropic < openai
	require.Equal(t, "claude-sonnet-4-5", entries[0].Name)
	require.Equal(t, "gpt-5", entries[1].Name)

	// claude 模型：跨渠道合并，分组按 ID 去重，不可访问的分组(11)被过滤
	claude := entries[0]
	require.Len(t, claude.Groups, 1)
	require.Equal(t, int64(10), claude.Groups[0].ID)
	require.NotNil(t, claude.Pricing)
	require.NotNil(t, claude.Pricing.InputPrice)
}

func TestBuildModelPlaza_UserFilterByAllowedGroups(t *testing.T) {
	allowed := map[int64]struct{}{11: {}} // 只可访问专属分组
	entries := buildModelPlaza(plazaTestChannels(), allowed)

	require.Len(t, entries, 1)
	require.Equal(t, "claude-sonnet-4-5", entries[0].Name)
	require.Len(t, entries[0].Groups, 1)
	require.Equal(t, int64(11), entries[0].Groups[0].ID)
	require.True(t, entries[0].Groups[0].IsExclusive)
}

func TestModelPlaza_Unauthenticated401(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := &AvailableChannelHandler{} // 401 路径不会触达 service 依赖

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/model-plaza", nil)

	h.ListModelPlaza(c)
	require.Equal(t, http.StatusUnauthorized, w.Code)
}


