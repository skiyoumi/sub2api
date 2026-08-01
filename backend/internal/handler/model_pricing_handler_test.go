//go:build unit

package handler

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestNormalizeModelPricingConfig(t *testing.T) {
	available := []service.PlazaGroup{{
		ID:     7,
		Models: []service.PlazaModel{{Name: "gpt-b"}, {Name: "gpt-a"}},
	}}
	config, ok := normalizeModelPricingConfig(&service.ModelPricingConfig{
		Enabled:     true,
		Description: "  pricing note  ",
		Groups:      []service.ModelPricingGroupConfig{{GroupID: 7, Models: []string{"gpt-b", "gpt-a", "gpt-b"}}},
	}, available)

	require.True(t, ok)
	require.Equal(t, "pricing note", config.Description)
	require.Equal(t, []string{"gpt-a", "gpt-b"}, config.Groups[0].Models)
}

func TestNormalizeModelPricingConfigRejectsUnknownModel(t *testing.T) {
	available := []service.PlazaGroup{{ID: 7, Models: []service.PlazaModel{{Name: "gpt-a"}}}}
	_, ok := normalizeModelPricingConfig(&service.ModelPricingConfig{
		Groups: []service.ModelPricingGroupConfig{{GroupID: 7, Models: []string{"missing"}}},
	}, available)
	require.False(t, ok)
}

func TestSelectPricingGroupsPreservesConfiguredGroupOrder(t *testing.T) {
	available := []service.PlazaGroup{
		{ID: 1, Models: []service.PlazaModel{{Name: "a"}, {Name: "b"}}},
		{ID: 2, Models: []service.PlazaModel{{Name: "c"}}},
	}
	selected := selectPricingGroups(available, []service.ModelPricingGroupConfig{
		{GroupID: 2, Models: []string{"c"}},
		{GroupID: 1, Models: []string{"b"}},
	})
	require.Equal(t, []int64{2, 1}, []int64{selected[0].ID, selected[1].ID})
	require.Equal(t, "b", selected[1].Models[0].Name)
}
