package service

import (
	"context"
	"encoding/json"
	"net"
	"strconv"
	"strings"
)

type registrationIPContextKey struct{}

func WithRegistrationIP(ctx context.Context, ip string) context.Context {
	return context.WithValue(ctx, registrationIPContextKey{}, strings.TrimSpace(ip))
}

func registrationIP(ctx context.Context) string {
	ip, _ := ctx.Value(registrationIPContextKey{}).(string)
	return strings.TrimSpace(ip)
}

func (s *SettingService) RegistrationIPLimit(ctx context.Context) int {
	if s == nil || s.settingRepo == nil { return 1 }
	value, err := s.settingRepo.GetValue(ctx, SettingKeyRegistrationIPLimit)
	if err != nil { return 1 }
	limit, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || limit < 0 { return 1 }
	return limit
}

func (s *SettingService) IsRegistrationIPWhitelisted(ctx context.Context, clientIP string) bool {
	if s == nil || s.settingRepo == nil { return false }
	value, err := s.settingRepo.GetValue(ctx, SettingKeyRegistrationIPWhitelist)
	if err != nil { return false }
	var entries []string
	if json.Unmarshal([]byte(value), &entries) != nil { return false }
	ip := net.ParseIP(strings.TrimSpace(clientIP))
	if ip == nil { return false }
	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		if parsed := net.ParseIP(entry); parsed != nil && parsed.Equal(ip) { return true }
		if _, network, err := net.ParseCIDR(entry); err == nil && network.Contains(ip) { return true }
	}
	return false
}
