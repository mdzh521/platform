package handlers

import (
	"fmt"
	"strings"
)

func deriveACKAccessMode(endpoint string, response ackClusterResponse) string {
	if strings.TrimSpace(endpoint) == "" {
		return "private-network"
	}
	if strings.Contains(endpoint, "private") || strings.TrimSpace(response.Endpoints.Private) == strings.TrimSpace(endpoint) {
		return "private-network"
	}
	return "direct"
}

func deriveEnrollmentStatus(providerStatus, endpoint string) string {
	switch strings.ToLower(strings.TrimSpace(providerStatus)) {
	case "active", "running":
		if strings.TrimSpace(endpoint) == "" {
			return "access_pending"
		}
		return "enrolled"
	case "creating", "initial", "scaling":
		return "created"
	default:
		return "error"
	}
}

func stringValue(value any) string {
	if value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return typed
	case *string:
		if typed == nil {
			return ""
		}
		return *typed
	default:
		return ""
	}
}

func flattenEnvMap(items map[string]string) []string {
	values := make([]string, 0, len(items))
	for key, value := range items {
		if strings.TrimSpace(key) == "" {
			continue
		}
		values = append(values, key+"="+value)
	}
	return values
}

func containsArg(items []string, expected string) bool {
	for _, item := range items {
		if item == expected {
			return true
		}
	}
	return false
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func firstNonEmptyStringSlice(values ...[]string) []string {
	for _, value := range values {
		items := compactStrings(value)
		if len(items) > 0 {
			return items
		}
	}
	return []string{}
}

func compactStrings(items []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(items))
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}

func appendUniqueString(items []string, value string) []string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return compactStrings(items)
	}
	return compactStrings(append(items, trimmed))
}

func extractStringSlice(value any) []string {
	switch typed := value.(type) {
	case []string:
		return compactStrings(typed)
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			if text := strings.TrimSpace(fmt.Sprint(item)); text != "" {
				out = append(out, text)
			}
		}
		return compactStrings(out)
	default:
		return []string{}
	}
}
