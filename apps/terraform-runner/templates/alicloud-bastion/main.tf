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
  region                = try(var.input.region, "cn-hangzhou")
  environment           = try(var.input.environment, "dev")
  bastion_name          = try(var.input.bastion_name, format("%s-%s", var.blueprint_code, local.environment))
  foundation_stack_name = try(var.input.foundation_stack_name, "")
  network_ref           = try(var.input.network_ref, "")
  name_prefix           = try(var.input.name_prefix, "")
  ops_refs              = try(var.input.ops_refs, [])
  provider_network_refs = try(var.input.provider_network_refs, {})
  vpc_id                = try(var.input.vpc_id, "")
  vswitch_id            = try(var.input.vswitch_id, "")
  instance_type         = try(var.input.instance_type, "ecs.t6-c1m2.large")
  instance_charge_type  = try(var.input.instance_charge_type, "PostPaid")
  period                = try(tonumber(var.input.period), 1)
  period_unit           = try(var.input.period_unit, "Month")
  ssh_key_name          = try(var.input.key_pair_name, "")
  system_disk_category  = try(var.input.system_disk_category, "cloud_essd")
  allocate_eip          = try(var.input.allocate_eip, true)
  ingress_cidrs         = try(var.input.ingress_cidrs, ["0.0.0.0/0"])
  default_tags          = tomap(try(var.input.default_tags, {}))
  sanitized_default_tags = {
    for key, value in local.default_tags :
    key => value
    if !contains(["name", "environment", "managedby", "blueprintcode", "role"], lower(key))
  }
  common_tags = merge(local.sanitized_default_tags, {
    Name          = local.bastion_name
    Environment   = local.environment
    ManagedBy     = "platform-center"
    BlueprintCode = var.blueprint_code
  })
  request = {
    provider              = var.cloud_provider
    account_id            = var.account_id
    blueprint_code        = var.blueprint_code
    network_plan_id       = var.network_plan_id
    action                = var.action
    region                = local.region
    bastion_name          = local.bastion_name
    instance_type         = local.instance_type
    instance_charge_type  = local.instance_charge_type
    period                = local.period
    period_unit           = local.period_unit
    system_disk_category  = local.system_disk_category
    environment           = local.environment
    network_ref           = local.network_ref
    foundation_stack_name = local.foundation_stack_name
    name_prefix           = local.name_prefix
    ops_refs              = local.ops_refs
    provider_network_refs = local.provider_network_refs
    vpc_id                = local.vpc_id
    vswitch_id            = local.vswitch_id
    allocate_eip          = local.allocate_eip
    ingress_cidrs         = local.ingress_cidrs
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

resource "alicloud_security_group" "bastion" {
  security_group_name = "${replace(local.bastion_name, "_", "-")}-sg"
  description         = "Bastion access security group"
  vpc_id              = local.vpc_id

  tags = merge(local.common_tags, {
    Name = "${local.bastion_name}-sg"
    Role = "bastion"
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
  security_group_id = alicloud_security_group.bastion.id
}

resource "alicloud_security_group_rule" "all_egress" {
  type              = "egress"
  ip_protocol       = "all"
  nic_type          = "intranet"
  policy            = "accept"
  port_range        = "-1/-1"
  cidr_ip           = "0.0.0.0/0"
  security_group_id = alicloud_security_group.bastion.id
}

resource "alicloud_instance" "bastion" {
  instance_name              = local.bastion_name
  host_name                  = replace(local.bastion_name, "_", "-")
  image_id                   = data.alicloud_images.alibaba_linux.images[0].id
  instance_type              = local.instance_type
  instance_charge_type       = local.instance_charge_type
  period                     = local.instance_charge_type == "PrePaid" ? local.period : null
  period_unit                = local.instance_charge_type == "PrePaid" ? local.period_unit : null
  vswitch_id                 = local.vswitch_id
  security_groups            = [alicloud_security_group.bastion.id]
  system_disk_category       = local.system_disk_category
  system_disk_size           = 40
  internet_max_bandwidth_out = 0
  key_name                   = local.ssh_key_name != "" ? local.ssh_key_name : null

  tags = merge(local.common_tags, {
    Name = local.bastion_name
    Role = "bastion"
  })
}

resource "alicloud_eip_address" "bastion" {
  count = local.allocate_eip ? 1 : 0
  depends_on = [
    alicloud_instance.bastion,
  ]

  address_name = "${replace(local.bastion_name, "_", "-")}-eip"
  isp          = "BGP"
  bandwidth    = "5"

  tags = merge(local.common_tags, {
    Name = "${local.bastion_name}-eip"
    Role = "bastion"
  })
}

resource "alicloud_eip_association" "bastion" {
  count = local.allocate_eip ? 1 : 0

  allocation_id = alicloud_eip_address.bastion[0].id
  instance_id   = alicloud_instance.bastion.id
  instance_type = "EcsInstance"
}

output "plan_context" {
  value = local.request
}

output "bastion_contract_summary" {
  value = {
    network_ref           = local.network_ref
    foundation_stack_name = local.foundation_stack_name
    ops_refs              = local.ops_refs
    provider_network_refs = local.provider_network_refs
    resolved_vpc_id       = local.vpc_id
    resolved_vswitch_id   = local.vswitch_id
  }
}

output "instance_id" {
  value = alicloud_instance.bastion.id
}

output "private_ip" {
  value = alicloud_instance.bastion.private_ip
}

output "public_ip" {
  value = local.allocate_eip ? alicloud_eip_address.bastion[0].ip_address : ""
}

output "ssh_endpoint" {
  value = local.allocate_eip ? format("ssh root@%s", alicloud_eip_address.bastion[0].ip_address) : ""
}

output "access_entry" {
  value = local.allocate_eip ? format("ssh root@%s", alicloud_eip_address.bastion[0].ip_address) : format("private:%s", alicloud_instance.bastion.private_ip)
}

output "security_group_id" {
  value = alicloud_security_group.bastion.id
}

output "vpc_id" {
  value = local.vpc_id
}

output "vswitch_id" {
  value = local.vswitch_id
}
