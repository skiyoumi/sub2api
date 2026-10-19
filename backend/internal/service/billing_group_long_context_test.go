//go:build unit

package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGroupModelLongContextPricing_BillingPriority(t *testing.T) {
	accountOn, accountOff := true, false
	cases := []struct {
		name            string
		groupEnabled    bool
		modelDisabled   bool
		accountEnabled  *bool
		wantLongContext bool
	}{
		{"disabled_with_group_on", true, true, nil, false},
		{"disabled_with_account_on", true, true, &accountOn, false},
		{"disabled_with_only_account_on", false, true, &accountOn, false},
		{"disabled_with_account_off", true, true, &accountOff, false},
		{"inherits_group_on", true, false, nil, true},
		{"inherits_group_off", false, false, nil, false},
		{"inherits_account_on", false, false, &accountOn, true},
		{"account_off_does_not_override_group", true, false, &accountOff, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			bs, resolver := newTokenCostTestEnv(t, PlatformOpenAI, nil, newStubPricingServiceFromJSON(t, openAILadderCatalogJSON))
			group := &Group{ID: 100, Platform: PlatformOpenAI, LongContextPricingEnabled: tc.groupEnabled,
				ModelPricing: []ChannelModelPricing{{Models: []string{"gpt-5.4"}, BillingMode: BillingModeToken, LongContextPricingDisabled: tc.modelDisabled}},
			}
			cost, err := bs.CalculateCostUnified(CostInput{
				Ctx: context.Background(), Model: "gpt-5.4", Group: group, GroupID: &group.ID, Resolver: resolver,
				Tokens:         UsageTokens{InputTokens: 300000, OutputTokens: 1000, CacheCreationTokens: 40000, CacheReadTokens: 20000},
				RateMultiplier: 1.5, LongContextBillingEnabled: tc.accountEnabled,
			})
			require.NoError(t, err)
			require.Equal(t, tc.wantLongContext, cost.LongContextBillingApplied)
			inputMultiplier, outputMultiplier := 1.0, 1.0
			if tc.wantLongContext {
				inputMultiplier, outputMultiplier = 2, 1.5
			}
			require.InDelta(t, 0.75*inputMultiplier, cost.InputCost, 1e-9)
			require.InDelta(t, 0.015*outputMultiplier, cost.OutputCost, 1e-9)
			require.InDelta(t, 0.1*inputMultiplier, cost.CacheCreationCost, 1e-9)
			require.InDelta(t, 0.005*inputMultiplier, cost.CacheReadCost, 1e-9)
			require.InDelta(t, (0.855*inputMultiplier+0.015*outputMultiplier)*1.5, cost.ActualCost, 1e-9)
		})
	}
}

func TestGroupModelLongContextPricing_MatchingAndCustomPrices(t *testing.T) {
	bs, resolver := newTokenCostTestEnv(t, PlatformOpenAI, nil, newStubPricingServiceFromJSON(t, openAILadderCatalogJSON))
	group := &Group{ID: 100, Platform: PlatformOpenAI, LongContextPricingEnabled: true,
		ModelPricing: []ChannelModelPricing{
			{Models: []string{"gpt-*"}, BillingMode: BillingModeToken, LongContextPricingDisabled: true, InputPrice: testPtrFloat64(10e-6), OutputPrice: testPtrFloat64(20e-6)},
			{Models: []string{"gpt-5.4"}, BillingMode: BillingModeToken},
		},
	}
	for _, tc := range []struct {
		name            string
		wantInputCost   float64
		wantOutputCost  float64
		wantLongContext bool
	}{
		{"gpt-5.4", 1.5, 0.0225, true},
		{"gpt-5.4-mini", 3, 0.02, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cost, err := bs.CalculateTokenCostForRequest(TokenCostRequest{
				Ctx: context.Background(), Model: tc.name, Group: group, Resolver: resolver,
				Tokens: UsageTokens{InputTokens: 300000, OutputTokens: 1000}, RateMultiplier: 1,
			})
			require.NoError(t, err)
			require.Equal(t, tc.wantLongContext, cost.LongContextBillingApplied)
			require.InDelta(t, tc.wantInputCost, cost.InputCost, 1e-9)
			require.InDelta(t, tc.wantOutputCost, cost.OutputCost, 1e-9)
		})
	}
}

func TestGroupModelLongContextPricing_PlazaMatchesBilling(t *testing.T) {
	channels := []Channel{{ID: 1, Name: "ch", Status: StatusActive, GroupIDs: []int64{10}, ModelPricing: []ChannelModelPricing{{
		Platform: PlatformOpenAI, Models: []string{"gpt-5.4", "gpt-5.4-pro"}, BillingMode: BillingModeToken,
	}}}}
	groups := []Group{{ID: 10, Name: "group", Platform: PlatformOpenAI, RateMultiplier: 1, LongContextPricingEnabled: true,
		ModelPricing: []ChannelModelPricing{{Models: []string{"gpt-5.4"}, BillingMode: BillingModeToken, LongContextPricingDisabled: true}},
	}}
	catalog := newStubPricingServiceFromJSON(t, openAILadderCatalogJSON)
	// Give another model the same catalog ladder to check that the exemption stays local.
	other := *catalog.pricingData["gpt-5.4"]
	catalog.pricingData["gpt-5.4-pro"] = &other
	svc := newPlazaServiceWithBilling(channels, groups, map[int64]string{10: PlatformOpenAI}, catalog)
	out, err := svc.ListGroups(context.Background())
	require.NoError(t, err)
	require.Len(t, out, 1)
	models := plazaModelsByName(out[0].Models)
	exempt := models["gpt-5.4"]
	require.Empty(t, exempt.Pricing.Intervals)
	require.Empty(t, exempt.LongContextBasis)
	require.InDelta(t, 2.5e-6, *exempt.Pricing.InputPrice, 1e-12)
	require.Len(t, exempt.OfficialPricing.Intervals, 2)
	require.Len(t, models["gpt-5.4-pro"].Pricing.Intervals, 2)
	cost, err := svc.billingService.CalculateTokenCostForRequest(TokenCostRequest{
		Ctx: context.Background(), Model: "gpt-5.4", Group: &groups[0], Resolver: svc.resolver,
		Tokens: UsageTokens{InputTokens: 300000}, RateMultiplier: 1,
	})
	require.NoError(t, err)
	require.False(t, cost.LongContextBillingApplied)
	require.InDelta(t, 300000**exempt.Pricing.InputPrice, cost.ActualCost, 1e-9)
}

func TestGroupModelLongContextPricing_SaveAndAuthCacheRoundtrip(t *testing.T) {
	repo := &groupRepoStubForAdmin{createID: 100}
	svc := &adminServiceImpl{groupRepo: repo}
	created, err := svc.CreateGroup(context.Background(), &CreateGroupInput{
		Name: "model-exemption", Platform: PlatformOpenAI, RateMultiplier: 1, LongContextPricingEnabled: true,
		ModelPricing: []ChannelModelPricing{{Models: []string{"gpt-5.4"}, BillingMode: BillingModeToken, LongContextPricingDisabled: true}},
	})
	require.NoError(t, err)
	require.True(t, repo.created.ModelPricing[0].LongContextPricingDisabled)
	groupID := created.ID
	apiKey := &APIKey{ID: 82, UserID: 40, GroupID: &groupID, Key: "sk-model-exemption", Status: StatusActive,
		User: &User{ID: 40, Status: StatusActive}, Group: created,
	}
	keySvc := &APIKeyService{}
	payload, err := json.Marshal(&APIKeyAuthCacheEntry{Snapshot: keySvc.snapshotFromAPIKey(context.Background(), apiKey)})
	require.NoError(t, err)
	var cached APIKeyAuthCacheEntry
	require.NoError(t, json.Unmarshal(payload, &cached))
	materialized, used, err := keySvc.applyAuthCacheEntry(apiKey.Key, &cached)
	require.NoError(t, err)
	require.True(t, used)
	require.True(t, materialized.Group.ModelPricing[0].LongContextPricingDisabled)

	repo.getByID = created
	pricing := []ChannelModelPricing{{Models: []string{"gpt-5.4"}, BillingMode: BillingModeToken, LongContextPricingDisabled: false}}
	updated, err := svc.UpdateGroup(context.Background(), groupID, &UpdateGroupInput{ModelPricing: &pricing})
	require.NoError(t, err)
	require.False(t, updated.ModelPricing[0].LongContextPricingDisabled)
	require.False(t, repo.updated.ModelPricing[0].LongContextPricingDisabled)
}
