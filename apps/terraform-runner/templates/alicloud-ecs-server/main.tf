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
  region                 = try(var.input.region, "cn-hangzhou")
  environment            = try(var.input.environment, "dev")
  project_name           = try(var.input.project_name, "")
  server_name            = try(var.input.server_name, format("%s-%s", var.blueprint_code, local.environment))
  server_count           = max(try(var.input.server_count, 1), 1)
  network_ref            = try(var.input.network_ref, "")
  foundation_stack_name  = try(var.input.foundation_stack_name, "")
  name_prefix            = try(var.input.name_prefix, "")
  workload_refs          = try(var.input.workload_refs, [])
  ops_refs               = try(var.input.ops_refs, [])
  provider_network_refs  = try(var.input.provider_network_refs, {})
  vpc_id                 = try(var.input.vpc_id, "")
  vswitch_id             = try(var.input.vswitch_id, "")
  subnet_id              = try(var.input.subnet_id, "")
  subnet_ids             = try(var.input.subnet_ids, [])
  server_instances_input = try(var.input.server_instances, [])
  instance_type          = try(var.input.instance_type, "ecs.g6.large")
  instance_charge_type   = try(var.input.instance_charge_type, "PostPaid")
  period                 = try(tonumber(var.input.period), 1)
  period_unit            = try(var.input.period_unit, "Month")
  system_disk_size_gb    = try(var.input.system_disk_size_gb, 40)
  system_disk_type       = try(var.input.system_disk_type, "cloud_essd")
  ssh_key_name           = try(var.input.key_pair_name, "")
  allocate_eip           = try(var.input.allocate_eip, true)
  ingress_cidrs          = try(var.input.ingress_cidrs, ["0.0.0.0/0"])
  login_username         = "root"
  default_tags           = tomap(try(var.input.default_tags, {}))
  sanitized_default_tags = {
    for key, value in local.default_tags :
    key => value
    if !contains(["name", "environment", "managedby", "blueprintcode", "role"], lower(key))
  }
  common_tags = merge(local.sanitized_default_tags, {
    Name          = local.server_name
    Environment   = local.environment
    ManagedBy     = "platform-center"
    BlueprintCode = var.blueprint_code
  })
  resource_name_parts   = regexall("[0-9a-z_=+,.@-]+", lower(local.server_name))
  resource_name         = length(local.resource_name_parts) > 0 ? join("-", local.resource_name_parts) : lower(var.blueprint_code)
  effective_vswitch_ids = length(local.subnet_ids) > 0 ? [for item in local.subnet_ids : tostring(item)] : compact([local.vswitch_id, local.subnet_id])
  default_server_instances = length(local.effective_vswitch_ids) == 0 ? [] : [
    for index in range(local.server_count) : {
      name       = local.server_count > 1 ? format("%s-%02d", local.server_name, index + 1) : local.server_name
      vswitch_id = local.effective_vswitch_ids[index % length(local.effective_vswitch_ids)]
    }
  ]
  normalized_server_instances = length(local.server_instances_input) > 0 ? [
    for index, item in local.server_instances_input : {
      name       = try(item.name, local.server_count > 1 ? format("%s-%02d", local.server_name, index + 1) : local.server_name)
      vswitch_id = try(item.vswitch_id, try(item.subnet_id, length(local.effective_vswitch_ids) > 0 ? local.effective_vswitch_ids[index % length(local.effective_vswitch_ids)] : local.vswitch_id))
    }
  ] : local.default_server_instances
  server_instances = [
    for item in local.normalized_server_instances : {
      name       = tostring(item.name)
      vswitch_id = tostring(item.vswitch_id)
    }
    if trimspace(tostring(try(item.name, ""))) != "" && trimspace(tostring(try(item.vswitch_id, ""))) != ""
  ]
  server_map = {
    for item in local.server_instances : item.name => item
  }
  primary_server_name = try(keys(local.server_map)[0], "")
  request = {
    provider              = var.cloud_provider
    account_id            = var.account_id
    blueprint_code        = var.blueprint_code
    network_plan_id       = var.network_plan_id
    action                = var.action
    region                = local.region
    network_ref           = local.network_ref
    foundation_stack_name = local.foundation_stack_name
    name_prefix           = local.name_prefix
    project_name          = local.project_name
    workload_refs         = local.workload_refs
    ops_refs              = local.ops_refs
    provider_network_refs = local.provider_network_refs
    server_name           = local.server_name
    server_count          = local.server_count
    instance_type         = local.instance_type
    instance_charge_type  = local.instance_charge_type
    period                = local.period
    period_unit           = local.period_unit
    system_disk_size_gb   = local.system_disk_size_gb
    system_disk_type      = local.system_disk_type
    vpc_id                = local.vpc_id
    vswitch_id            = local.vswitch_id
    subnet_ids            = local.subnet_ids
    server_instances      = local.server_instances
  }
}

provider "alicloud" {
  region = local.region
}

data "alicloud_images" "alibaba_linux" {
  owners      = "system"
  most_recent = true
  name_regex  = "^aliyun_3_.*_x64.*"
}

resource "alicloud_security_group" "server" {
  security_group_name = "${local.resource_name}-sg"
  description         = "Application server access security group"
  vpc_id              = local.vpc_id

  tags = merge(local.common_tags, {
    Name = "${local.resource_name}-sg"
  })
}

resource "alicloud_security_group_rule" "ssh_ingress" {
  count = length(local.ingress_cidrs)

  type              = "ingress"
  ip_protocol       = "tcp"
  nic_type          = "intranet"
  policy            = "accept"
  port_range        = "22/22"
  cidr_ip           = local.ingress_cidrs[count.index]
  security_group_id = alicloud_security_group.server.id
}

resource "alicloud_security_group_rule" "all_egress" {
  type              = "egress"
  ip_protocol       = "all"
  nic_type          = "intranet"
  policy            = "accept"
  port_range        = "-1/-1"
  cidr_ip           = "0.0.0.0/0"
  security_group_id = alicloud_security_group.server.id
}

resource "alicloud_instance" "server" {
  for_each = local.server_map

  instance_name              = each.key
  host_name                  = replace(each.key, "_", "-")
  image_id                   = data.alicloud_images.alibaba_linux.images[0].id
  instance_type              = local.instance_type
  instance_charge_type       = local.instance_charge_type
  period                     = local.instance_charge_type == "PrePaid" ? local.period : null
  period_unit                = local.instance_charge_type == "PrePaid" ? local.period_unit : null
  vswitch_id                 = each.value.vswitch_id
  security_groups            = [alicloud_security_group.server.id]
  system_disk_category       = local.system_disk_type
  system_disk_size           = local.system_disk_size_gb
  internet_max_bandwidth_out = 0
  key_name                   = local.ssh_key_name != "" ? local.ssh_key_name : null

  tags = merge(local.common_tags, {
    Name = each.key
    Role = "server"
  })
}

resource "alicloud_eip_address" "server" {
  for_each = local.allocate_eip ? local.server_map : {}
  depends_on = [
    alicloud_instance.server,
  ]

  address_name = "${replace(each.key, "_", "-")}-eip"
  isp          = "BGP"
  bandwidth    = "5"

  tags = merge(local.common_tags, {
    Name = "${each.key}-eip"
    Role = "server-eip"
  })
}

resource "alicloud_eip_association" "server" {
  for_each = local.allocate_eip ? local.server_map : {}

  allocation_id = alicloud_eip_address.server[each.key].id
  instance_id   = alicloud_instance.server[each.key].id
  instance_type = "EcsInstance"
}

output "plan_context" {
  value = local.request
}

output "instance_id" {
  value = try(alicloud_instance.server[local.primary_server_name].id, "")
}

output "private_ip" {
  value = try(alicloud_instance.server[local.primary_server_name].private_ip, "")
}

output "public_ip" {
  value = local.allocate_eip ? try(alicloud_eip_address.server[local.primary_server_name].ip_address, "") : ""
}

output "ssh_endpoint" {
  value = local.allocate_eip ? format("ssh %s@%s", local.login_username, try(alicloud_eip_address.server[local.primary_server_name].ip_address, "")) : format("private:%s", try(alicloud_instance.server[local.primary_server_name].private_ip, ""))
}

output "access_entry" {
  value = local.allocate_eip ? format("ssh %s@%s", local.login_username, try(alicloud_eip_address.server[local.primary_server_name].ip_address, "")) : format("private:%s", try(alicloud_instance.server[local.primary_server_name].private_ip, ""))
}

output "security_group_id" {
  value = alicloud_security_group.server.id
}

output "vpc_id" {
  value = local.vpc_id
}

output "vswitch_id" {
  value = try(local.server_map[local.primary_server_name].vswitch_id, local.vswitch_id)
}

output "instance_ids" {
  value = {
    for name, item in alicloud_instance.server : name => item.id
  }
}

output "private_ips" {
  value = {
    for name, item in alicloud_instance.server : name => item.private_ip
  }
}

output "public_ips" {
  value = {
    for name, item in alicloud_eip_address.server : name => item.ip_address
  }
}

output "server_instances" {
  value = {
    for name, item in alicloud_instance.server : name => {
      id             = item.id
      name           = name
      private_ip     = item.private_ip
      public_ip      = local.allocate_eip ? try(alicloud_eip_address.server[name].ip_address, "") : ""
      vswitch_id     = local.server_map[name].vswitch_id
      login_username = local.login_username
    }
  }
}

output "access_entries" {
  value = {
    for name, item in alicloud_instance.server : name => {
      instance_id   = item.id
      private_ip    = item.private_ip
      public_ip     = local.allocate_eip ? try(alicloud_eip_address.server[name].ip_address, "") : ""
      login_user    = local.login_username
      access_entry  = local.allocate_eip ? format("ssh %s@%s", local.login_username, try(alicloud_eip_address.server[name].ip_address, "")) : format("private:%s", item.private_ip)
      key_pair_name = local.ssh_key_name
    }
  }
}
