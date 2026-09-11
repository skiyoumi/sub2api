package service

import (
	"encoding/json"
	"fmt"
	"regexp"
)

// SidebarMenuOrder orders existing top-level menu paths within each sidebar section.
// Unknown paths are ignored by the client; this setting never grants menu access.
type SidebarMenuOrder struct {
	User  []string `json:"user"`
	Admin []string `json:"admin,omitempty"`
}

func ParseSidebarMenuOrder(raw string) SidebarMenuOrder {
	var order SidebarMenuOrder
	if json.Unmarshal([]byte(raw), &order) != nil {
		return SidebarMenuOrder{User: []string{}}
	}
	if order.User == nil {
		order.User = []string{}
	}
	return order
}

func (o SidebarMenuOrder) Public() SidebarMenuOrder {
	return SidebarMenuOrder{User: o.User}
}

var sidebarMenuPathPattern = regexp.MustCompile(`^/[a-zA-Z0-9_-]+(/[a-zA-Z0-9_-]+)*$`)

func (o SidebarMenuOrder) Validate() error {
	for _, section := range []struct {
		name  string
		paths []string
	}{{"user", o.User}, {"admin", o.Admin}} {
		if len(section.paths) > 100 {
			return fmt.Errorf("sidebar_menu_order.%s supports at most 100 entries", section.name)
		}
		seen := make(map[string]bool, len(section.paths))
		for _, path := range section.paths {
			if len(path) > 128 || !sidebarMenuPathPattern.MatchString(path) {
				return fmt.Errorf("sidebar_menu_order.%s contains an invalid menu path", section.name)
			}
			if seen[path] {
				return fmt.Errorf("sidebar_menu_order.%s contains a duplicate menu path", section.name)
			}
			seen[path] = true
		}
	}
	return nil
}
