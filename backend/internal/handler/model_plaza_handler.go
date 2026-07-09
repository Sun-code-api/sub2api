package handler

import (
	"sort"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// 模型广场：以「模型」为中心聚合所有活跃渠道的定价与分组倍率，供用户端模型广场页展示。
//
// 数据完全复用 ChannelService.ListAvailable 的展示链路（含 LiteLLM 全局价格回落），
// 不触碰真实计费逻辑。可见性与「可用渠道」页保持一致：按用户可访问分组过滤，
// 受 available-channels 功能开关控制（关闭时返回空列表），
// 用户专属倍率由前端走 /groups/rates 合并。

// modelPlazaEntry 模型广场单个模型条目。
type modelPlazaEntry struct {
	Name     string                     `json:"name"`
	Platform string                     `json:"platform"`
	Pricing  *userSupportedModelPricing `json:"pricing"`
	Groups   []userAvailableGroup       `json:"groups"`
}

// ListModelPlaza 登录版模型广场：按当前用户可访问分组过滤。
// GET /api/v1/model-plaza
func (h *AvailableChannelHandler) ListModelPlaza(c *gin.Context) {
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	if !h.featureEnabled(c) {
		response.Success(c, []modelPlazaEntry{})
		return
	}

	userGroups, err := h.apiKeyService.GetAvailableGroups(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	allowedGroupIDs := make(map[int64]struct{}, len(userGroups))
	for i := range userGroups {
		allowedGroupIDs[userGroups[i].ID] = struct{}{}
	}

	channels, err := h.channelService.ListAvailable(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, buildModelPlaza(channels, allowedGroupIDs))
}

// buildModelPlaza 把渠道视图聚合为模型为中心的条目列表，仅保留 allowedGroupIDs 中的分组。
// 同名同平台模型跨渠道合并：定价取首个非空（各渠道未配置时均已回落到全局
// LiteLLM 数据，展示上等价），分组按 ID 去重后并集。
func buildModelPlaza(
	channels []service.AvailableChannel,
	allowedGroupIDs map[int64]struct{},
) []modelPlazaEntry {
	type plazaKey struct{ name, platform string }
	byKey := make(map[plazaKey]*modelPlazaEntry)
	seenGroups := make(map[plazaKey]map[int64]struct{})

	for i := range channels {
		ch := &channels[i]
		if ch.Status != service.StatusActive {
			continue
		}

		groupsByPlatform := make(map[string][]userAvailableGroup, 4)
		for _, g := range ch.Groups {
			if g.Platform == "" {
				continue
			}
			if _, ok := allowedGroupIDs[g.ID]; !ok {
				continue
			}
			groupsByPlatform[g.Platform] = append(groupsByPlatform[g.Platform], userAvailableGroup{
				ID:                 g.ID,
				Name:               g.Name,
				Platform:           g.Platform,
				SubscriptionType:   g.SubscriptionType,
				RateMultiplier:     g.RateMultiplier,
				PeakRateEnabled:    g.PeakRateEnabled,
				PeakStart:          g.PeakStart,
				PeakEnd:            g.PeakEnd,
				PeakRateMultiplier: g.PeakRateMultiplier,
				IsExclusive:        g.IsExclusive,
			})
		}
		if len(groupsByPlatform) == 0 {
			continue
		}

		for j := range ch.SupportedModels {
			m := &ch.SupportedModels[j]
			platformGroups := groupsByPlatform[m.Platform]
			if len(platformGroups) == 0 {
				continue
			}
			key := plazaKey{name: m.Name, platform: m.Platform}
			entry, ok := byKey[key]
			if !ok {
				entry = &modelPlazaEntry{
					Name:     m.Name,
					Platform: m.Platform,
					Pricing:  toUserPricing(m.Pricing),
				}
				byKey[key] = entry
				seenGroups[key] = make(map[int64]struct{})
			}
			if entry.Pricing == nil && m.Pricing != nil {
				entry.Pricing = toUserPricing(m.Pricing)
			}
			for _, g := range platformGroups {
				if _, dup := seenGroups[key][g.ID]; dup {
					continue
				}
				seenGroups[key][g.ID] = struct{}{}
				entry.Groups = append(entry.Groups, g)
			}
		}
	}

	out := make([]modelPlazaEntry, 0, len(byKey))
	for _, entry := range byKey {
		sort.SliceStable(entry.Groups, func(i, j int) bool {
			return entry.Groups[i].Name < entry.Groups[j].Name
		})
		out = append(out, *entry)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Platform != out[j].Platform {
			return out[i].Platform < out[j].Platform
		}
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	return out
}
