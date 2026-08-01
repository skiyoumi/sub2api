package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

const SettingKeyModelPricingConfig = "model_pricing_config"

type ModelPricingGroupConfig struct {
	GroupID int64    `json:"group_id"`
	Models  []string `json:"models"`
}

type ModelPricingConfig struct {
	Enabled     bool                      `json:"enabled"`
	Description string                    `json:"description"`
	Groups      []ModelPricingGroupConfig `json:"groups"`
}

func DefaultModelPricingConfig() *ModelPricingConfig {
	return &ModelPricingConfig{Groups: []ModelPricingGroupConfig{}}
}

func (s *SettingService) GetModelPricingConfig(ctx context.Context) (*ModelPricingConfig, error) {
	value, err := s.settingRepo.GetValue(ctx, SettingKeyModelPricingConfig)
	if err != nil {
		if errors.Is(err, ErrSettingNotFound) {
			return DefaultModelPricingConfig(), nil
		}
		return nil, fmt.Errorf("get model pricing config: %w", err)
	}
	if value == "" {
		return DefaultModelPricingConfig(), nil
	}
	var config ModelPricingConfig
	if err := json.Unmarshal([]byte(value), &config); err != nil {
		return nil, fmt.Errorf("decode model pricing config: %w", err)
	}
	if config.Groups == nil {
		config.Groups = []ModelPricingGroupConfig{}
	}
	return &config, nil
}

func (s *SettingService) SetModelPricingConfig(ctx context.Context, config *ModelPricingConfig) error {
	if config == nil {
		return fmt.Errorf("model pricing config cannot be nil")
	}
	data, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("encode model pricing config: %w", err)
	}
	if err := s.settingRepo.Set(ctx, SettingKeyModelPricingConfig, string(data)); err != nil {
		return fmt.Errorf("save model pricing config: %w", err)
	}
	if s.onUpdate != nil {
		s.onUpdate()
	}
	return nil
}
