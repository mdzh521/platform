terraform {
  required_version = ">= 1.6.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

locals {
  region                  = try(var.input.region, "ap-southeast-1")
  environment             = try(var.input.environment, "dev")
  vpc_name                = try(var.input.vpc_name, format("%s-%s", var.blueprint_code, local.environment))
  vpc_cidr                = try(var.input.vpc_cidr, "10.10.0.0/16")
  availability_zone_count = max(1, tonumber(try(var.input.availability_zone_count, 2)))
  public_subnets          = try(var.input.public_subnets, [])
  private_subnets         = try(var.input.private_subnets, [])
  nat_gateway_count       = max(0, tonumber(try(var.input.nat_gateway_count, 1)))
  create_bastion_subnet   = try(var.input.create_bastion_subnet, false)
  default_tags            = tomap(try(var.input.default_tags, {}))
  az_letters              = ["a", "b", "c", "d", "e", "f"]
  az_names                = [for suffix in slice(local.az_letters, 0, min(length(local.az_letters), local.availability_zone_count)) : "${local.region}${suffix}"]
  bastion_subnet_cidr     = try(var.input.bastion_subnet_cidr, "")

  normalized_subnet_groups = [
    for group in try(var.input.subnet_groups, []) : {
      role        = lower(tostring(try(group.role, "application")))
      tier        = lower(tostring(try(group.tier, "private")))
      description = tostring(try(group.description, tostring(try(group.role, "application"))))
      name_prefix = tostring(try(group.name_prefix, format("%s-%s", local.vpc_name, tostring(try(group.role, "application")))))
      cidrs       = [for cidr in try(group.cidrs, []) : tostring(cidr)]
    }
    if length(try(group.cidrs, [])) > 0
  ]

  fallback_public_groups = length(local.public_subnets) > 0 ? [
    {
      role        = "ingress"
      tier        = "public"
      description = "公网入口"
      name_prefix = format("%s-ingress", local.vpc_name)
      cidrs       = local.public_subnets
    }
  ] : []

  fallback_private_groups = length(local.private_subnets) > 0 ? [
    {
      role        = "application"
      tier        = "private"
      description = "应用内网"
      name_prefix = format("%s-application", local.vpc_name)
      cidrs       = local.private_subnets
    }
  ] : []

  fallback_bastion_groups = local.create_bastion_subnet && local.bastion_subnet_cidr != "" ? [
    {
      role        = "bastion"
      tier        = "private"
      description = "运维跳板"
      name_prefix = format("%s-bastion", local.vpc_name)
      cidrs       = [local.bastion_subnet_cidr]
    }
  ] : []

  effective_subnet_groups = length(local.normalized_subnet_groups) > 0 ? local.normalized_subnet_groups : concat(local.fallback_public_groups, local.fallback_private_groups, local.fallback_bastion_groups)

  common_tags = merge(local.default_tags, {
    Name          = local.vpc_name
    Environment   = local.environment
    ManagedBy     = "platform-center"
    BlueprintCode = var.blueprint_code
  })

  public_subnet_entries = flatten([
    for group in local.effective_subnet_groups : [
      for idx, cidr in group.cidrs : {
        key         = format("%s-%02d", group.role, idx + 1)
        cidr        = cidr
        az          = local.az_names[idx % length(local.az_names)]
        role        = group.role
        description = group.description
        name        = format("%s-%02d", group.name_prefix, idx + 1)
      }
    ] if group.tier == "public"
  ])

  private_subnet_entries = flatten([
    for group in local.effective_subnet_groups : [
      for idx, cidr in group.cidrs : {
        key         = format("%s-%02d", group.role, idx + 1)
        cidr        = cidr
        az          = local.az_names[idx % length(local.az_names)]
        role        = group.role
        description = group.description
        nat_index   = local.nat_gateway_count > 0 ? min(idx, local.nat_gateway_count - 1) : null
        name        = format("%s-%02d", group.name_prefix, idx + 1)
      }
    ] if group.tier != "public"
  ])

  public_subnet_map = { for item in local.public_subnet_entries : item.key => item }
  private_subnet_map = { for item in local.private_subnet_entries : item.key => item }

  nat_gateway_map = {
    for idx in range(min(local.nat_gateway_count, length(local.public_subnet_entries))) :
    format("nat-%02d", idx + 1) => {
      public_key = local.public_subnet_entries[idx].key
      name       = format("%s-nat-%02d", local.vpc_name, idx + 1)
    }
  }

  request = {
    provider                 = var.cloud_provider
    account_id               = var.account_id
    blueprint_code           = var.blueprint_code
    network_plan_id          = var.network_plan_id
    action                   = var.action
    region                   = local.region
    vpc_name                 = local.vpc_name
    vpc_cidr                 = local.vpc_cidr
    availability_zone_count  = local.availability_zone_count
    public_subnet_count      = length(local.public_subnet_entries)
    private_subnet_count     = length(local.private_subnet_entries)
    subnet_group_count       = length(local.effective_subnet_groups)
    nat_gateway_count        = local.nat_gateway_count
    create_bastion_subnet    = local.create_bastion_subnet
    environment              = local.environment
  }
}

provider "aws" {
  region                      = local.region
  skip_credentials_validation = true
  skip_metadata_api_check     = true
  skip_requesting_account_id  = true
  skip_region_validation      = true
}

resource "aws_vpc" "this" {
  cidr_block           = local.vpc_cidr
  enable_dns_support   = true
  enable_dns_hostnames = true

  tags = merge(local.common_tags, {
    Name = local.vpc_name
  })
}

resource "aws_internet_gateway" "this" {
  vpc_id = aws_vpc.this.id

  tags = merge(local.common_tags, {
    Name = "${local.vpc_name}-igw"
  })
}

resource "aws_subnet" "public" {
  for_each = local.public_subnet_map

  vpc_id                  = aws_vpc.this.id
  cidr_block              = each.value.cidr
  availability_zone       = each.value.az
  map_public_ip_on_launch = true

  tags = merge(local.common_tags, {
    Name = each.value.name
    Tier = "public"
    Role = each.value.role
  })
}

resource "aws_subnet" "private" {
  for_each = local.private_subnet_map

  vpc_id            = aws_vpc.this.id
  cidr_block        = each.value.cidr
  availability_zone = each.value.az

  tags = merge(local.common_tags, {
    Name = each.value.name
    Tier = "private"
    Role = each.value.role
  })
}

resource "aws_eip" "nat" {
  for_each = local.nat_gateway_map

  domain = "vpc"

  tags = merge(local.common_tags, {
    Name = "${each.value.name}-eip"
  })
}

resource "aws_nat_gateway" "this" {
  for_each = local.nat_gateway_map

  allocation_id = aws_eip.nat[each.key].id
  subnet_id     = aws_subnet.public[each.value.public_key].id

  tags = merge(local.common_tags, {
    Name = each.value.name
  })

  depends_on = [aws_internet_gateway.this]
}

resource "aws_route_table" "public" {
  vpc_id = aws_vpc.this.id

  route {
    cidr_block = "0.0.0.0/0"
    gateway_id = aws_internet_gateway.this.id
  }

  tags = merge(local.common_tags, {
    Name = "${local.vpc_name}-public-rt"
    Tier = "public"
  })
}

resource "aws_route_table_association" "public" {
  for_each = aws_subnet.public

  subnet_id      = each.value.id
  route_table_id = aws_route_table.public.id
}

resource "aws_route_table" "private" {
  for_each = aws_subnet.private

  vpc_id = aws_vpc.this.id

  dynamic "route" {
    for_each = local.nat_gateway_count > 0 ? [1] : []
    content {
      cidr_block     = "0.0.0.0/0"
      nat_gateway_id = aws_nat_gateway.this[format("nat-%02d", local.private_subnet_map[each.key].nat_index + 1)].id
    }
  }

  tags = merge(local.common_tags, {
    Name = "${local.private_subnet_map[each.key].name}-rt"
    Tier = "private"
  })
}

resource "aws_route_table_association" "private" {
  for_each = aws_subnet.private

  subnet_id      = each.value.id
  route_table_id = aws_route_table.private[each.key].id
}

output "plan_context" {
  value = local.request
}

output "vpc_id" {
  value = aws_vpc.this.id
}

output "public_subnet_ids" {
  value = { for key, subnet in aws_subnet.public : key => subnet.id }
}

output "private_subnet_ids" {
  value = { for key, subnet in aws_subnet.private : key => subnet.id }
}

output "internet_gateway_id" {
  value = aws_internet_gateway.this.id
}

output "nat_gateway_ids" {
  value = { for key, gateway in aws_nat_gateway.this : key => gateway.id }
}
