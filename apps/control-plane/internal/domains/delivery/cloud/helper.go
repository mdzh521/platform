package cloud

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

func buildTopology(input NetworkPlanInput) map[string]any {
	availabilityZones := input.AvailabilityZones
	if availabilityZones <= 0 {
		availabilityZones = 2
	}
	natCount := input.NATGatewayCount
	if natCount < 0 {
		natCount = 0
	}

	environment := normalizeField(input.Environment, "dev")
	vpcName := normalizeField(input.VPCName, deriveVPCName(input.Name, environment))
	provider := normalizeProvider(input.Provider)
	subnetGroups := normalizeSubnetGroups(input, vpcName)
	roleRefs := buildNetworkRoleRefs(subnetGroups)

	topology := map[string]any{
		"foundation_stack_name":   vpcName,
		"name_prefix":             vpcName,
		"environment":             environment,
		"vpc_name":                vpcName,
		"availability_zones":      availabilityZones,
		"availability_zone_count": availabilityZones,
		"subnet_groups":           subnetGroups,
		"network_role_refs":       roleRefs,
		"internet_entry_refs":     collectRoleRefs(roleRefs, "internet-entry"),
		"egress_refs":             collectRoleRefs(roleRefs, "egress"),
		"workload_refs":           collectRoleRefs(roleRefs, "workload"),
		"data_refs":               collectRoleRefs(roleRefs, "data"),
		"ops_refs":                collectRoleRefs(roleRefs, "ops"),
		"cluster_node_refs":       collectRoleRefs(roleRefs, "cluster-node"),
		"pod_network_refs":        collectRoleRefs(roleRefs, "pod-network"),
		"provider_network_refs":   buildProviderNetworkRefs(input.Provider, roleRefs),
		"default_tags":            cleanStringList(input.DefaultTags),
		"nat_gateway_count":       natCount,
		"create_bastion_subnet":   input.CreateBastion,
		"security_baseline":       normalizeField(input.SecurityBaseline, "standard"),
	}
	if provider == "aws" {
		topology["public_subnets"] = flattenSubnetCIDRs(subnetGroups, "public")
		topology["private_subnets"] = flattenSubnetCIDRs(subnetGroups, "private")
	} else {
		// compatibility only; alicloud primary semantics are role-based vSwitch refs
		topology["public_subnets"] = flattenTrafficProfileCIDRs(subnetGroups, "internet-entry")
		topology["private_subnets"] = flattenNonTrafficProfileCIDRs(subnetGroups, "internet-entry")
	}
	if bastionCIDR := strings.TrimSpace(input.BastionSubnetCIDR); bastionCIDR != "" {
		topology["bastion_subnet_cidr"] = bastionCIDR
	}
	return topology
}

func normalizeSubnetGroups(input NetworkPlanInput, vpcName string) []map[string]any {
	groups := make([]map[string]any, 0)
	if len(input.SubnetGroups) > 0 {
		for _, group := range input.SubnetGroups {
			cidrs := cleanStringList(group.CIDRs)
			if len(cidrs) == 0 {
				continue
			}
			role := normalizeSubnetRole(group.Role)
			tier, trafficProfile := normalizeGroupSemantics(input.Provider, role, group.Tier, group.TrafficProfile)
			groups = append(groups, map[string]any{
				"key":             role,
				"role":            role,
				"tier":            tier,
				"traffic_profile": trafficProfile,
				"description":     normalizeField(group.Description, role),
				"name_prefix":     fmt.Sprintf("%s-%s", vpcName, role),
				"cidrs":           cidrs,
			})
		}
	}
	if len(groups) > 0 {
		return groups
	}

	publicSubnets := firstNonEmptyList(cleanStringList(input.PublicSubnets), splitCSV(input.PublicSubnetCIDR))
	if len(publicSubnets) > 0 {
		_, trafficProfile := normalizeGroupSemantics(input.Provider, "ingress", "public", "")
		groups = append(groups, map[string]any{
			"key":             "ingress",
			"role":            "ingress",
			"tier":            "public",
			"traffic_profile": trafficProfile,
			"description":     "公网入口",
			"name_prefix":     fmt.Sprintf("%s-ingress", vpcName),
			"cidrs":           publicSubnets,
		})
	}
	privateSubnets := firstNonEmptyList(cleanStringList(input.PrivateSubnets), splitCSV(input.PrivateSubnetCIDR))
	if len(privateSubnets) > 0 {
		_, trafficProfile := normalizeGroupSemantics(input.Provider, "application", "private", "")
		groups = append(groups, map[string]any{
			"key":             "application",
			"role":            "application",
			"tier":            "private",
			"traffic_profile": trafficProfile,
			"description":     "应用内网",
			"name_prefix":     fmt.Sprintf("%s-application", vpcName),
			"cidrs":           privateSubnets,
		})
	}
	if input.CreateBastion && strings.TrimSpace(input.BastionSubnetCIDR) != "" {
		tier, trafficProfile := normalizeGroupSemantics(input.Provider, "bastion", "private", "")
		groups = append(groups, map[string]any{
			"key":             "bastion",
			"role":            "bastion",
			"tier":            tier,
			"traffic_profile": trafficProfile,
			"description":     "运维跳板",
			"name_prefix":     fmt.Sprintf("%s-bastion", vpcName),
			"cidrs":           []string{strings.TrimSpace(input.BastionSubnetCIDR)},
		})
	}
	return groups
}

func flattenSubnetCIDRs(groups []map[string]any, tier string) []string {
	result := []string{}
	for _, group := range groups {
		if strings.TrimSpace(fmt.Sprint(group["tier"])) != tier {
			continue
		}
		switch typed := group["cidrs"].(type) {
		case []string:
			result = append(result, typed...)
		case []any:
			for _, item := range typed {
				cidr := strings.TrimSpace(fmt.Sprint(item))
				if cidr != "" {
					result = append(result, cidr)
				}
			}
		}
	}
	return result
}

func flattenTrafficProfileCIDRs(groups []map[string]any, profile string) []string {
	result := []string{}
	for _, group := range groups {
		if strings.TrimSpace(fmt.Sprint(group["traffic_profile"])) != profile {
			continue
		}
		result = append(result, extractStringSlice(group["cidrs"])...)
	}
	return result
}

func flattenNonTrafficProfileCIDRs(groups []map[string]any, excluded string) []string {
	result := []string{}
	for _, group := range groups {
		if strings.TrimSpace(fmt.Sprint(group["traffic_profile"])) == excluded {
			continue
		}
		result = append(result, extractStringSlice(group["cidrs"])...)
	}
	return result
}

func buildNetworkRoleRefs(groups []map[string]any) map[string]any {
	result := map[string]any{}
	for _, group := range groups {
		role := strings.TrimSpace(fmt.Sprint(group["role"]))
		if role == "" {
			continue
		}
		cidrs := extractStringSlice(group["cidrs"])
		namePrefix := strings.TrimSpace(fmt.Sprint(group["name_prefix"]))
		plannedNames := make([]string, 0, len(cidrs))
		for index := range cidrs {
			plannedNames = append(plannedNames, fmt.Sprintf("%s-%02d", namePrefix, index+1))
		}
		result[role] = map[string]any{
			"role":            role,
			"tier":            strings.TrimSpace(fmt.Sprint(group["tier"])),
			"traffic_profile": normalizeField(strings.TrimSpace(fmt.Sprint(group["traffic_profile"])), roleTrafficProfile(role)),
			"name_prefix":     namePrefix,
			"planned_names":   plannedNames,
			"cidrs":           cidrs,
			"description":     strings.TrimSpace(fmt.Sprint(group["description"])),
		}
	}
	return result
}

func collectRoleRefs(roleRefs map[string]any, profile string) []map[string]any {
	result := []map[string]any{}
	for _, value := range roleRefs {
		row, ok := value.(map[string]any)
		if !ok {
			continue
		}
		if strings.TrimSpace(fmt.Sprint(row["traffic_profile"])) != profile {
			continue
		}
		result = append(result, row)
	}
	return result
}

func buildProviderNetworkRefs(provider string, roleRefs map[string]any) map[string]any {
	refs := map[string]any{}
	switch normalizeProvider(provider) {
	case "alicloud":
		refs["slb_vswitch_refs"] = collectNamedRoleRefs(roleRefs, "slb")
		refs["application_vswitch_refs"] = collectNamedRoleRefs(roleRefs, "application", "middleware", "etl")
		refs["database_vswitch_refs"] = collectNamedRoleRefs(roleRefs, "database", "warehouse", "cache")
		refs["ops_vswitch_refs"] = collectNamedRoleRefs(roleRefs, "ops", "bastion")
		refs["ack_node_vswitch_refs"] = collectNamedRoleRefs(roleRefs, "ack-node")
		refs["ack_pod_vswitch_refs"] = collectNamedRoleRefs(roleRefs, "ack-pod")
	default:
		refs["public_subnet_refs"] = collectTierRoleRefs(roleRefs, "public")
		refs["private_subnet_refs"] = collectTierRoleRefs(roleRefs, "private")
		refs["ingress_subnet_refs"] = collectNamedRoleRefs(roleRefs, "ingress")
		refs["workload_subnet_refs"] = collectNamedRoleRefs(roleRefs, "application", "middleware", "k8s", "etl")
		refs["data_subnet_refs"] = collectNamedRoleRefs(roleRefs, "database", "warehouse", "cache")
		refs["ops_subnet_refs"] = collectNamedRoleRefs(roleRefs, "ops", "bastion")
	}
	return refs
}

func collectNamedRoleRefs(roleRefs map[string]any, roles ...string) []map[string]any {
	wanted := map[string]struct{}{}
	for _, role := range roles {
		role = strings.TrimSpace(role)
		if role != "" {
			wanted[role] = struct{}{}
		}
	}
	result := []map[string]any{}
	for key, value := range roleRefs {
		if _, ok := wanted[key]; !ok {
			continue
		}
		if row, ok := value.(map[string]any); ok {
			result = append(result, row)
		}
	}
	return result
}

func collectTierRoleRefs(roleRefs map[string]any, tier string) []map[string]any {
	result := []map[string]any{}
	for _, value := range roleRefs {
		row, ok := value.(map[string]any)
		if !ok {
			continue
		}
		if strings.TrimSpace(fmt.Sprint(row["tier"])) != tier {
			continue
		}
		result = append(result, row)
	}
	return result
}

func extractStringSlice(value any) []string {
	switch typed := value.(type) {
	case []string:
		return cleanStringList(typed)
	case []any:
		result := make([]string, 0, len(typed))
		for _, item := range typed {
			text := strings.TrimSpace(fmt.Sprint(item))
			if text != "" {
				result = append(result, text)
			}
		}
		return result
	default:
		return []string{}
	}
}

func roleTrafficProfile(role string) string {
	switch strings.TrimSpace(strings.ToLower(role)) {
	case "ingress", "slb", "elb", "alb", "nlb":
		return "internet-entry"
	case "nat", "egress":
		return "egress"
	case "database", "warehouse", "cache":
		return "data"
	case "ops", "bastion":
		return "ops"
	case "ack-node", "cluster-node", "k8s-node":
		return "cluster-node"
	case "ack-pod", "pod-network", "k8s-pod":
		return "pod-network"
	default:
		return "workload"
	}
}

func normalizeGroupSemantics(provider, role, tier, trafficProfile string) (string, string) {
	provider = normalizeProvider(provider)
	role = strings.TrimSpace(strings.ToLower(role))
	tier = normalizeSubnetTier(tier)
	trafficProfile = normalizeTrafficProfile(strings.TrimSpace(strings.ToLower(trafficProfile)))
	if trafficProfile == "" {
		trafficProfile = roleTrafficProfile(role)
		if provider == "aws" && tier == "public" && trafficProfile == "workload" {
			trafficProfile = "internet-entry"
		}
	}
	if provider == "aws" {
		if tier == "" {
			switch trafficProfile {
			case "internet-entry", "egress":
				tier = "public"
			default:
				tier = "private"
			}
		}
		return tier, trafficProfile
	}
	// alicloud keeps role-based vSwitch as primary expression; tier is compatibility only
	return "", trafficProfile
}

func normalizeTrafficProfile(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "internet-entry", "egress", "workload", "data", "ops", "cluster-node", "pod-network":
		return strings.TrimSpace(strings.ToLower(value))
	default:
		return ""
	}
}

func normalizeProvider(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	switch value {
	case "aws", "alicloud":
		return value
	default:
		return ""
	}
}

func normalizeField(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func parseStringList(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return []string{}
	}
	var items []string
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return []string{}
	}
	return items
}

func cleanStringList(items []string) []string {
	result := make([]string, 0, len(items))
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func splitCSV(value string) []string {
	parts := strings.Split(strings.TrimSpace(value), ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func firstNonEmptyList(primary, fallback []string) []string {
	if len(primary) > 0 {
		return primary
	}
	return fallback
}

func mustJSON(value any) []byte {
	raw, _ := json.Marshal(value)
	return raw
}

func mustJSONString(value any) string {
	return string(mustJSON(value))
}

func networkLabel(id *uint, items map[uint]string) string {
	if id == nil {
		return ""
	}
	return items[*id]
}

func deriveVPCName(name, environment string) string {
	base := slugify(name)
	if base == "" {
		base = "platform-network"
	}
	return fmt.Sprintf("%s-%s", base, environment)
}

func normalizeSubnetRole(value string) string {
	role := slugify(value)
	if role == "" {
		return "application"
	}
	return role
}

func normalizeSubnetTier(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "public":
		return "public"
	default:
		return "private"
	}
}

var slugPattern = regexp.MustCompile(`[^a-z0-9]+`)

func slugify(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	normalized = slugPattern.ReplaceAllString(normalized, "-")
	return strings.Trim(normalized, "-")
}
