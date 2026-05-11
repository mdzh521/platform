terraform {
  required_version = ">= 1.6.0"

  required_providers {
    alicloud = {
      source  = "aliyun/alicloud"
      version = "~> 1.0"
    }
  }
}

locals {
  region                  = try(var.input.region, "cn-hangzhou")
  environment             = try(var.input.environment, "dev")
  vpc_name                = try(var.input.vpc_name, format("%s-%s", var.blueprint_code, local.environment))
  vpc_cidr                = try(var.input.vpc_cidr, "10.20.0.0/16")
  availability_zone_count = max(1, tonumber(try(var.input.availability_zone_count, 2)))
  network_role_refs       = try(var.input.network_role_refs, {})
  public_subnets          = try(var.input.public_subnets, [])
  private_subnets         = try(var.input.private_subnets, [])
  create_bastion_subnet   = try(var.input.create_bastion_subnet, false)
  bastion_subnet_cidr     = try(var.input.bastion_subnet_cidr, "10.20.100.0/24")
  nat_gateway_count       = max(0, tonumber(try(var.input.nat_gateway_count, 1)))
  default_tags            = tomap(try(var.input.default_tags, {}))

  normalized_subnet_groups = [
    for group in try(var.input.subnet_groups, []) : {
      role            = lower(tostring(try(group.role, "application")))
      tier            = lower(tostring(try(group.tier, "")))
      traffic_profile = lower(tostring(try(group.traffic_profile, "")))
      description     = tostring(try(group.description, tostring(try(group.role, "application"))))
      name_prefix     = tostring(try(group.name_prefix, format("%s-%s", local.vpc_name, tostring(try(group.role, "application")))))
      cidrs           = [for cidr in try(group.cidrs, []) : tostring(cidr)]
    }
    if length(try(group.cidrs, [])) > 0
  ]

  zone_ids = slice(
    [for zone in data.alicloud_zones.available.zones : zone.id],
    0,
    min(length(data.alicloud_zones.available.zones), local.availability_zone_count)
  )

  fallback_public_groups = length(local.public_subnets) > 0 ? [
    {
      role            = "slb"
      tier            = ""
      traffic_profile = "internet-entry"
      description     = "入口交换机"
      name_prefix     = format("%s-slb", local.vpc_name)
      cidrs           = local.public_subnets
    }
  ] : []

  fallback_private_groups = length(local.private_subnets) > 0 ? [
    {
      role            = "application"
      tier            = ""
      traffic_profile = "workload"
      description     = "应用交换机"
      name_prefix     = format("%s-application", local.vpc_name)
      cidrs           = local.private_subnets
    }
  ] : []

  fallback_bastion_groups = local.create_bastion_subnet && local.bastion_subnet_cidr != "" ? [
    {
      role            = "ops"
      tier            = ""
      traffic_profile = "ops"
      description     = "运维接入交换机"
      name_prefix     = format("%s-ops", local.vpc_name)
      cidrs           = [local.bastion_subnet_cidr]
    }
  ] : []

  effective_subnet_groups = length(local.normalized_subnet_groups) > 0 ? local.normalized_subnet_groups : concat(local.fallback_public_groups, local.fallback_private_groups, local.fallback_bastion_groups)

  common_tags = merge(local.default_tags, {
    Name          = local.vpc_name
    Environment   = local.environment
    ManagedBy     = "platform-center"
    BlueprintCode = var.blueprint_code
  })

  vswitch_entries = flatten([
    for group in local.effective_subnet_groups : [
      for idx, cidr in group.cidrs : {
        key             = format("%s-%02d", group.role, idx + 1)
        cidr            = cidr
        zone            = local.zone_ids[idx % length(local.zone_ids)]
        role            = group.role
        traffic_profile = group.traffic_profile != "" ? group.traffic_profile : (
          group.role == "slb" ? "internet-entry" :
          group.role == "ack-node" ? "cluster-node" :
          group.role == "ack-pod" ? "pod-network" :
          contains(["database", "warehouse", "cache"], group.role) ? "data" :
          contains(["ops", "bastion"], group.role) ? "ops" :
          "workload"
        )
        description = group.description
        name        = format("%s-%02d", group.name_prefix, idx + 1)
      }
    ]
  ])

  vswitch_map = { for item in local.vswitch_entries : item.key => item }

  request = {
    provider                = var.cloud_provider
    account_id              = var.account_id
    blueprint_code          = var.blueprint_code
    network_plan_id         = var.network_plan_id
    action                  = var.action
    region                  = local.region
    vpc_name                = local.vpc_name
    vpc_cidr                = local.vpc_cidr
    availability_zone_count = local.availability_zone_count
    network_role_refs       = local.network_role_refs
    vswitch_count           = length(local.vswitch_entries)
    subnet_group_count      = length(local.effective_subnet_groups)
    nat_gateway_count       = local.nat_gateway_count
    create_bastion_subnet   = local.create_bastion_subnet
    environment             = local.environment
  }
}

provider "alicloud" {
  region = local.region
}

data "alicloud_zones" "available" {
  available_resource_creation = "VSwitch"
}

resource "alicloud_vpc" "this" {
  cidr_block = local.vpc_cidr
  vpc_name   = local.vpc_name

  tags = local.common_tags
}

resource "alicloud_vswitch" "role_based" {
  for_each = local.vswitch_map

  vpc_id       = alicloud_vpc.this.id
  cidr_block   = each.value.cidr
  zone_id      = each.value.zone
  vswitch_name = each.value.name

  tags = merge(local.common_tags, {
    Name           = each.value.name
    Role           = each.value.role
    TrafficProfile = each.value.traffic_profile
  })
}

output "plan_context" {
  value = local.request
}

output "vpc_id" {
  value = alicloud_vpc.this.id
}

output "slb_vswitch_ids" {
  value = { for key, subnet in alicloud_vswitch.role_based : key => subnet.id if try(subnet.tags["Role"], "") == "slb" }
}

output "application_vswitch_ids" {
  value = { for key, subnet in alicloud_vswitch.role_based : key => subnet.id if contains(["application", "middleware", "etl"], try(subnet.tags["Role"], "")) }
}

output "database_vswitch_ids" {
  value = { for key, subnet in alicloud_vswitch.role_based : key => subnet.id if contains(["database", "warehouse", "cache"], try(subnet.tags["Role"], "")) }
}

output "ops_vswitch_ids" {
  value = { for key, subnet in alicloud_vswitch.role_based : key => subnet.id if contains(["ops", "bastion"], try(subnet.tags["Role"], "")) }
}

output "ack_node_vswitch_ids" {
  value = { for key, subnet in alicloud_vswitch.role_based : key => subnet.id if try(subnet.tags["Role"], "") == "ack-node" }
}

output "ack_pod_vswitch_ids" {
  value = { for key, subnet in alicloud_vswitch.role_based : key => subnet.id if try(subnet.tags["Role"], "") == "ack-pod" }
}
