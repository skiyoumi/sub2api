package handler

import (
	"sort"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

type modelPricingAdminResponse struct {
	Config *service.ModelPricingConfig `json:"config"`
	Groups []modelPlazaGroup           `json:"groups"`
}

func (h *ModelPlazaHandler) GetPricing(c *gin.Context) {
	config, err := h.settingService.GetModelPricingConfig(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	groups, err := h.channelService.ListPlazaGroups(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	subject, authed := middleware.GetAuthSubjectFromContext(c)
	if !authed {
		response.Unauthorized(c, "Authentication required")
		return
	}
	allowedExclusive, err := h.apiKeyService.GetUserAllowedGroupIDSet(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	userRates, _ := h.apiKeyService.GetUserGroupRates(c.Request.Context(), subject.UserID)
	selected := []service.PlazaGroup{}
	if config.Enabled {
		selected = selectPricingGroups(filterPlazaVisibleGroups(groups, allowedExclusive), config.Groups)
	}
	out := make([]modelPlazaGroup, 0, len(selected))
	for i := range selected {
		out = append(out, toModelPlazaGroupDTO(&selected[i], userRates))
	}
	response.Success(c, modelPlazaResponse{Description: config.Description, Groups: out})
}

func (h *ModelPlazaHandler) GetPricingConfig(c *gin.Context) {
	config, err := h.settingService.GetModelPricingConfig(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	groups, err := h.channelService.ListPlazaGroups(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	out := make([]modelPlazaGroup, 0, len(groups))
	for i := range groups {
		out = append(out, toModelPlazaGroupDTO(&groups[i], nil))
	}
	response.Success(c, modelPricingAdminResponse{Config: config, Groups: out})
}

func (h *ModelPlazaHandler) UpdatePricingConfig(c *gin.Context) {
	var req service.ModelPricingConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request body: "+err.Error())
		return
	}
	if len(req.Description) > 4000 {
		response.BadRequest(c, "Description must not exceed 4000 characters")
		return
	}
	available, err := h.channelService.ListPlazaGroups(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	normalized, ok := normalizeModelPricingConfig(&req, available)
	if !ok {
		response.BadRequest(c, "Configuration contains an unavailable group or model")
		return
	}
	if err := h.settingService.SetModelPricingConfig(c.Request.Context(), normalized); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, normalized)
}

func normalizeModelPricingConfig(config *service.ModelPricingConfig, available []service.PlazaGroup) (*service.ModelPricingConfig, bool) {
	byID := make(map[int64]service.PlazaGroup, len(available))
	for _, group := range available {
		byID[group.ID] = group
	}
	seenGroups := make(map[int64]struct{}, len(config.Groups))
	result := &service.ModelPricingConfig{Enabled: config.Enabled, Description: strings.TrimSpace(config.Description), Groups: make([]service.ModelPricingGroupConfig, 0, len(config.Groups))}
	for _, selected := range config.Groups {
		group, exists := byID[selected.GroupID]
		if !exists {
			return nil, false
		}
		if _, duplicate := seenGroups[selected.GroupID]; duplicate {
			return nil, false
		}
		seenGroups[selected.GroupID] = struct{}{}
		availableModels := make(map[string]struct{}, len(group.Models))
		for _, model := range group.Models {
			availableModels[model.Name] = struct{}{}
		}
		seenModels := make(map[string]struct{}, len(selected.Models))
		models := make([]string, 0, len(selected.Models))
		for _, rawName := range selected.Models {
			name := strings.TrimSpace(rawName)
			if _, exists := availableModels[name]; !exists {
				return nil, false
			}
			if _, duplicate := seenModels[name]; duplicate {
				continue
			}
			seenModels[name] = struct{}{}
			models = append(models, name)
		}
		if len(models) == 0 {
			continue
		}
		sort.Strings(models)
		result.Groups = append(result.Groups, service.ModelPricingGroupConfig{GroupID: selected.GroupID, Models: models})
	}
	return result, true
}

func selectPricingGroups(available []service.PlazaGroup, selected []service.ModelPricingGroupConfig) []service.PlazaGroup {
	byID := make(map[int64]service.PlazaGroup, len(available))
	for _, group := range available {
		byID[group.ID] = group
	}
	result := make([]service.PlazaGroup, 0, len(selected))
	for _, selection := range selected {
		group, exists := byID[selection.GroupID]
		if !exists {
			continue
		}
		modelSet := make(map[string]struct{}, len(selection.Models))
		for _, name := range selection.Models {
			modelSet[name] = struct{}{}
		}
		models := make([]service.PlazaModel, 0, len(selection.Models))
		for _, model := range group.Models {
			if _, exists := modelSet[model.Name]; exists {
				models = append(models, model)
			}
		}
		if len(models) == 0 {
			continue
		}
		group.Models = models
		result = append(result, group)
	}
	return result
}
