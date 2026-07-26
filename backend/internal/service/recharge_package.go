package service

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/shopspring/decimal"
)

const maxRechargePackages = 20
const maxRechargePackagesJSONBytes = 64 * 1024

var rechargePackageIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)

// RechargePackage is an administrator-defined balance recharge offer. Amounts
// remain decimal strings in settings so configuration never passes through a
// binary floating-point representation.
type RechargePackage struct {
	ID                string `json:"id"`
	Amount            string `json:"amount"`
	BonusAmount       string `json:"bonus_amount"`
	BonusValidityDays int    `json:"bonus_validity_days"`
	Recommended       bool   `json:"recommended"`
	Enabled           bool   `json:"enabled"`
	SortOrder         int    `json:"sort_order"`
}

type RechargePackageSelection struct {
	Package         RechargePackage
	BaseAmount      decimal.Decimal
	PermanentAmount decimal.Decimal
	BonusAmount     decimal.Decimal
	CreditedAmount  decimal.Decimal
	ConfigHash      string
}

func ResolveRechargePackage(cfg *PaymentConfig, packageID string, requestedAmount float64) (*RechargePackageSelection, error) {
	packageID = strings.TrimSpace(packageID)
	if packageID == "" {
		if cfg != nil && cfg.RechargePackagesEnabled && !cfg.AllowCustomRecharge {
			return nil, infraerrors.Forbidden("CUSTOM_RECHARGE_DISABLED", "custom recharge amount is disabled")
		}
		return nil, nil
	}
	if cfg == nil || !cfg.RechargePackagesEnabled {
		return nil, infraerrors.BadRequest("RECHARGE_PACKAGE_NOT_FOUND", "recharge package not found")
	}
	var selected *RechargePackage
	for i := range cfg.RechargePackages {
		if cfg.RechargePackages[i].ID == packageID {
			selected = &cfg.RechargePackages[i]
			break
		}
	}
	if selected == nil {
		return nil, infraerrors.BadRequest("RECHARGE_PACKAGE_NOT_FOUND", "recharge package not found")
	}
	if !selected.Enabled {
		return nil, infraerrors.Conflict("RECHARGE_PACKAGE_DISABLED", "recharge package is disabled")
	}
	base, _ := decimal.NewFromString(selected.Amount)
	if !decimal.NewFromFloat(requestedAmount).Equal(base) {
		return nil, infraerrors.Conflict("RECHARGE_PACKAGE_CHANGED", "recharge package has changed")
	}
	bonus, _ := decimal.NewFromString(selected.BonusAmount)
	permanent := base.Mul(decimal.NewFromFloat(normalizeBalanceRechargeMultiplier(cfg.BalanceRechargeMultiplier))).Round(2)
	encoded, _ := json.Marshal(selected)
	hash := fmt.Sprintf("%x", sha256.Sum256(encoded))
	return &RechargePackageSelection{
		Package: *selected, BaseAmount: base, PermanentAmount: permanent,
		BonusAmount: bonus, CreditedAmount: permanent.Add(bonus), ConfigHash: hash,
	}, nil
}

func (s *RechargePackageSelection) Snapshot() map[string]any {
	if s == nil {
		return nil
	}
	return map[string]any{
		"id": s.Package.ID, "amount": s.BaseAmount.StringFixed(2),
		"permanent_credit_amount": s.PermanentAmount.String(),
		"bonus_amount":            s.BonusAmount.String(),
		"bonus_validity_days":     s.Package.BonusValidityDays,
		"recommended":             s.Package.Recommended, "config_hash": s.ConfigHash,
	}
}

func RechargePackageSelectionFromProviderSnapshot(snapshot map[string]any) (*RechargePackageSelection, error) {
	if snapshot == nil {
		return nil, nil
	}
	raw, ok := snapshot["_recharge_package"]
	if !ok {
		return nil, nil
	}
	encoded, err := json.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("encode recharge package snapshot: %w", err)
	}
	var stored struct {
		ID                    string `json:"id"`
		Amount                string `json:"amount"`
		PermanentCreditAmount string `json:"permanent_credit_amount"`
		BonusAmount           string `json:"bonus_amount"`
		BonusValidityDays     int    `json:"bonus_validity_days"`
		ConfigHash            string `json:"config_hash"`
	}
	if err := json.Unmarshal(encoded, &stored); err != nil {
		return nil, fmt.Errorf("decode recharge package snapshot: %w", err)
	}
	base, err := decimal.NewFromString(stored.Amount)
	if err != nil {
		return nil, fmt.Errorf("invalid recharge package base amount snapshot")
	}
	permanent, err := decimal.NewFromString(stored.PermanentCreditAmount)
	if err != nil {
		return nil, fmt.Errorf("invalid recharge package permanent amount snapshot")
	}
	bonus, err := decimal.NewFromString(stored.BonusAmount)
	if err != nil {
		return nil, fmt.Errorf("invalid recharge package bonus amount snapshot")
	}
	return &RechargePackageSelection{
		Package:    RechargePackage{ID: stored.ID, Amount: stored.Amount, BonusAmount: stored.BonusAmount, BonusValidityDays: stored.BonusValidityDays, Enabled: true},
		BaseAmount: base, PermanentAmount: permanent, BonusAmount: bonus,
		CreditedAmount: permanent.Add(bonus), ConfigHash: stored.ConfigHash,
	}, nil
}

func (p *RechargePackage) UnmarshalJSON(data []byte) error {
	var value struct {
		ID                string          `json:"id"`
		Amount            json.RawMessage `json:"amount"`
		BonusAmount       json.RawMessage `json:"bonus_amount"`
		BonusValidityDays int             `json:"bonus_validity_days"`
		Recommended       bool            `json:"recommended"`
		Enabled           bool            `json:"enabled"`
		SortOrder         int             `json:"sort_order"`
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&value); err != nil {
		return err
	}
	amount, err := decodeRechargePackageDecimal(value.Amount)
	if err != nil {
		return fmt.Errorf("amount: %w", err)
	}
	bonusAmount, err := decodeRechargePackageDecimal(value.BonusAmount)
	if err != nil {
		return fmt.Errorf("bonus_amount: %w", err)
	}
	*p = RechargePackage{
		ID: value.ID, Amount: amount, BonusAmount: bonusAmount,
		BonusValidityDays: value.BonusValidityDays, Recommended: value.Recommended,
		Enabled: value.Enabled, SortOrder: value.SortOrder,
	}
	return nil
}

func decodeRechargePackageDecimal(raw json.RawMessage) (string, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return "", nil
	}
	if trimmed[0] == '"' {
		var value string
		if err := json.Unmarshal(trimmed, &value); err != nil {
			return "", err
		}
		return value, nil
	}
	var value json.Number
	if err := json.Unmarshal(trimmed, &value); err != nil {
		return "", fmt.Errorf("must be a number or decimal string")
	}
	return value.String(), nil
}

// ParseRechargePackages validates and normalizes the complete configuration.
func ParseRechargePackages(raw string, enabled bool, minAmount, maxAmount float64) ([]RechargePackage, error) {
	if len(raw) > maxRechargePackagesJSONBytes {
		return nil, fmt.Errorf("recharge package configuration exceeds %d bytes", maxRechargePackagesJSONBytes)
	}
	if strings.TrimSpace(raw) == "" {
		if enabled {
			return nil, fmt.Errorf("at least one recharge package is required")
		}
		return []RechargePackage{}, nil
	}
	var packages []RechargePackage
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&packages); err != nil {
		return nil, fmt.Errorf("decode recharge packages: %w", err)
	}
	if len(packages) > maxRechargePackages || (enabled && len(packages) == 0) {
		return nil, fmt.Errorf("recharge packages must contain between 1 and %d items", maxRechargePackages)
	}

	ids := make(map[string]struct{}, len(packages))
	enabledAmounts := make(map[string]struct{}, len(packages))
	for i := range packages {
		p := &packages[i]
		if !rechargePackageIDPattern.MatchString(p.ID) {
			return nil, fmt.Errorf("package %d has an invalid id", i+1)
		}
		if _, exists := ids[p.ID]; exists {
			return nil, fmt.Errorf("package id %q is duplicated", p.ID)
		}
		ids[p.ID] = struct{}{}

		amount, err := parsePackageAmount(p.Amount, 2, false)
		if err != nil {
			return nil, fmt.Errorf("package %q amount: %w", p.ID, err)
		}
		bonus, err := parsePackageAmount(p.BonusAmount, 8, true)
		if err != nil {
			return nil, fmt.Errorf("package %q bonus_amount: %w", p.ID, err)
		}
		if minAmount > 0 && amount.LessThan(decimal.NewFromFloat(minAmount)) {
			return nil, fmt.Errorf("package %q amount is below the global minimum", p.ID)
		}
		if maxAmount > 0 && amount.GreaterThan(decimal.NewFromFloat(maxAmount)) {
			return nil, fmt.Errorf("package %q amount exceeds the global maximum", p.ID)
		}
		if bonus.IsPositive() && (p.BonusValidityDays < 1 || p.BonusValidityDays > 3650) {
			return nil, fmt.Errorf("package %q bonus_validity_days must be between 1 and 3650", p.ID)
		}
		if bonus.IsZero() && p.BonusValidityDays != 0 {
			return nil, fmt.Errorf("package %q bonus_validity_days must be zero without a bonus", p.ID)
		}
		if p.SortOrder < 0 || p.SortOrder > 100000 {
			return nil, fmt.Errorf("package %q sort_order must be between 0 and 100000", p.ID)
		}
		p.Amount = amount.StringFixed(2)
		p.BonusAmount = bonus.String()
		if p.Enabled {
			key := amount.StringFixed(2)
			if _, exists := enabledAmounts[key]; exists {
				return nil, fmt.Errorf("enabled package amount %s is duplicated", key)
			}
			enabledAmounts[key] = struct{}{}
		}
	}
	sort.SliceStable(packages, func(i, j int) bool {
		if packages[i].SortOrder == packages[j].SortOrder {
			return packages[i].ID < packages[j].ID
		}
		return packages[i].SortOrder < packages[j].SortOrder
	})
	return packages, nil
}

func parsePackageAmount(raw string, maxScale int32, allowZero bool) (decimal.Decimal, error) {
	value, err := decimal.NewFromString(strings.TrimSpace(raw))
	if err != nil {
		return decimal.Zero, fmt.Errorf("must be a decimal string")
	}
	if value.IsNegative() || (!allowZero && value.IsZero()) {
		if allowZero {
			return decimal.Zero, fmt.Errorf("must be non-negative")
		}
		return decimal.Zero, fmt.Errorf("must be positive")
	}
	if -value.Exponent() > maxScale {
		return decimal.Zero, fmt.Errorf("allows at most %d decimal places", maxScale)
	}
	return value, nil
}

func enabledRechargePackages(packages []RechargePackage) []RechargePackage {
	result := make([]RechargePackage, 0, len(packages))
	for _, p := range packages {
		if p.Enabled {
			result = append(result, p)
		}
	}
	return result
}
