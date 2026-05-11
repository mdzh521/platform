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
  region                 = try(var.input.region, "cn-shenzhen")
  environment            = try(var.input.environment, "dev")
  cluster_name           = try(var.input.cluster_name, format("%s-%s", var.blueprint_code, local.environment))
  kubernetes_version     = try(var.input.kubernetes_version, "1.33.3-aliyun.1")
  cluster_spec           = try(var.input.cluster_spec, "ack.pro.small")
  worker_instance_type   = try(var.input.worker_instance_type, "ecs.g6.large")
  desired_capacity       = try(tonumber(var.input.desired_capacity), 2)
  min_size               = try(tonumber(var.input.min_size), 1)
  max_size               = try(tonumber(var.input.max_size), max(local.desired_capacity, 2))
  node_system_disk_category = try(var.input.node_system_disk_category, "cloud_essd")
  node_system_disk_size     = try(tonumber(var.input.node_system_disk_size), 120)
  node_login_password       = try(var.input.node_login_password, "")
  key_pair_name             = try(var.input.key_pair_name, "")
  network_ref               = try(var.input.network_ref, "")
  foundation_stack_name     = try(var.input.foundation_stack_name, "")
  cluster_node_refs         = try(var.input.cluster_node_refs, [])
  pod_network_refs          = try(var.input.pod_network_refs, [])
  provider_network_refs     = try(var.input.provider_network_refs, {})
  ops_refs                  = try(var.input.ops_refs, [])
  endpoint_access           = lower(try(var.input.endpoint_access, "private"))
  vpc_id                    = try(var.input.vpc_id, "")
  subnet_ids                = distinct(compact(try(var.input.subnet_ids, [])))
  pod_vswitch_ids           = distinct(compact(try(var.input.pod_vswitch_ids, [])))
  slb_vswitch_ids           = distinct(compact(try(var.input.slb_vswitch_ids, [])))
  service_cidr              = try(var.input.service_cidr, "172.21.0.0/20")
  pod_cidr                  = try(var.input.pod_cidr, "172.20.0.0/16")
  create_ack_service_roles  = try(tobool(var.input.create_ack_service_roles), true)
  ack_service_roles         = try(var.input.ack_service_roles, {
    AliyunCSDefaultRole               = "AliyunCSDefaultRolePolicy"
    AliyunCSManagedKubernetesRole     = "AliyunCSManagedKubernetesRolePolicy"
    AliyunCSManagedCmsRole            = "AliyunCSManagedCmsRolePolicy"
    AliyunCSManagedLogRole            = "AliyunCSManagedLogRolePolicy"
    AliyunCSManagedArmsRole           = "AliyunCSManagedArmsRolePolicy"
    AliyunCSManagedCsiRole            = "AliyunCSManagedCsiRolePolicy"
    AliyunCSManagedCsiProvisionerRole = "AliyunCSManagedCsiProvisionerRolePolicy"
    AliyunCSManagedCsiPluginRole      = "AliyunCSManagedCsiPluginRolePolicy"
    AliyunCSServerlessKubernetesRole  = "AliyunCSServerlessKubernetesRolePolicy"
    AliyunCSKubernetesAuditRole       = "AliyunCSKubernetesAuditRolePolicy"
    AliyunCSManagedNetworkRole        = "AliyunCSManagedNetworkRolePolicy"
    AliyunCISDefaultRole              = "AliyunCISDefaultRolePolicy"
    AliyunOOSLifecycleHook4CSRole     = "AliyunOOSLifecycleHook4CSRolePolicy"
  })
  cluster_addon_profile     = lower(try(var.input.cluster_addon_profile, "standard"))
  install_ebs_csi           = try(tobool(var.input.install_ebs_csi), true)
  install_efs_csi           = try(tobool(var.input.install_efs_csi), true)
  install_lb_controller     = try(tobool(var.input.install_lb_controller), true)
  install_metrics_server    = try(tobool(var.input.install_metrics_server), false)
  default_tags              = tomap(try(var.input.default_tags, {}))
  sanitized_default_tags = {
    for key, value in local.default_tags :
    key => value
    if !contains(["name", "environment", "managedby", "blueprintcode", "role"], lower(key))
  }
  common_tags = merge(local.sanitized_default_tags, {
    Name          = local.cluster_name
    Environment   = local.environment
    ManagedBy     = "platform-center"
    BlueprintCode = var.blueprint_code
  })
  request = {
    provider                 = var.cloud_provider
    account_id               = var.account_id
    blueprint_code           = var.blueprint_code
    network_plan_id          = var.network_plan_id
    action                   = var.action
    region                   = local.region
    cluster_name             = local.cluster_name
    kubernetes_version       = local.kubernetes_version
    cluster_spec             = local.cluster_spec
    worker_instance_type     = local.worker_instance_type
    desired_capacity         = local.desired_capacity
    min_size                 = local.min_size
    max_size                 = local.max_size
    network_ref              = local.network_ref
    foundation_stack_name    = local.foundation_stack_name
    cluster_node_refs        = local.cluster_node_refs
    pod_network_refs         = local.pod_network_refs
    provider_network_refs    = local.provider_network_refs
    ops_refs                 = local.ops_refs
    endpoint_access          = local.endpoint_access
    vpc_id                   = local.vpc_id
    subnet_ids               = local.subnet_ids
    pod_vswitch_ids          = local.pod_vswitch_ids
    slb_vswitch_ids          = local.slb_vswitch_ids
    service_cidr             = local.service_cidr
    pod_cidr                 = local.pod_cidr
    cluster_addon_profile    = local.cluster_addon_profile
    install_ebs_csi          = local.install_ebs_csi
    install_efs_csi          = local.install_efs_csi
    install_lb_controller    = local.install_lb_controller
    install_metrics_server   = local.install_metrics_server
    create_ack_service_roles = local.create_ack_service_roles
  }
}

provider "alicloud" {
  region = local.region
}

data "alicloud_ack_service" "this" {
  enable = "On"
  type   = "propayasgo"
}

resource "alicloud_ram_role" "ack_service_roles" {
  for_each = local.create_ack_service_roles ? local.ack_service_roles : {}

  role_name = each.key
  assume_role_policy_document = jsonencode({
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Principal = {
          Service = [
            contains(["AliyunOOSLifecycleHook4CSRole"], each.key) ? "oos.aliyuncs.com" : "cs.aliyuncs.com"
          ]
        }
      }
    ]
    Version = "1"
  })
  description = "Platform-managed ACK service role ${each.key}"
  force       = true
}

resource "alicloud_ram_role_policy_attachment" "ack_service_roles" {
  for_each = local.create_ack_service_roles ? local.ack_service_roles : {}

  policy_name = each.value
  policy_type = "System"
  role_name   = alicloud_ram_role.ack_service_roles[each.key].role_name
}

resource "alicloud_cs_managed_kubernetes" "this" {
  name                 = local.cluster_name
  cluster_spec         = local.cluster_spec
  version              = local.kubernetes_version
  vswitch_ids          = local.subnet_ids
  new_nat_gateway      = true
  pod_cidr             = local.pod_cidr
  service_cidr         = local.service_cidr
  slb_internet_enabled = local.endpoint_access == "public"
  deletion_protection  = false
  enable_rrsa          = true
  tags                 = local.common_tags

  depends_on = [
    data.alicloud_ack_service.this,
    alicloud_ram_role_policy_attachment.ack_service_roles,
  ]
}

resource "alicloud_cs_kubernetes_node_pool" "default" {
  cluster_id            = alicloud_cs_managed_kubernetes.this.id
  node_pool_name        = "${local.cluster_name}-default"
  vswitch_ids           = local.subnet_ids
  instance_types        = [local.worker_instance_type]
  desired_size          = local.desired_capacity
  password              = local.node_login_password != "" ? local.node_login_password : null
  key_name              = local.key_pair_name != "" ? local.key_pair_name : null
  install_cloud_monitor = true

  system_disk_category = local.node_system_disk_category
  system_disk_size     = local.node_system_disk_size
  runtime_name         = "containerd"
  image_type           = "AliyunLinux3"
}

output "plan_context" {
  value = local.request
}

output "cluster_id" {
  value = alicloud_cs_managed_kubernetes.this.id
}

output "cluster_name" {
  value = alicloud_cs_managed_kubernetes.this.name
}

output "cluster_endpoint" {
  value = try(alicloud_cs_managed_kubernetes.this.connections.api_server_internet, "")
}

output "cluster_status" {
  value = "created"
}

output "node_pool_id" {
  value = alicloud_cs_kubernetes_node_pool.default.id
}

output "node_pool_name" {
  value = alicloud_cs_kubernetes_node_pool.default.node_pool_name
}

output "vpc_id" {
  value = local.vpc_id
}

output "worker_vswitch_ids" {
  value = local.subnet_ids
}

output "pod_vswitch_ids" {
  value = local.pod_vswitch_ids
}

output "slb_vswitch_ids" {
  value = local.slb_vswitch_ids
}

output "cluster_contract_summary" {
  value = {
    network_ref           = local.network_ref
    foundation_stack_name = local.foundation_stack_name
    cluster_node_refs     = local.cluster_node_refs
    pod_network_refs      = local.pod_network_refs
    provider_network_refs = local.provider_network_refs
    ops_refs              = local.ops_refs
    vpc_id                = local.vpc_id
    worker_vswitch_ids    = local.subnet_ids
    pod_vswitch_ids       = local.pod_vswitch_ids
    slb_vswitch_ids       = local.slb_vswitch_ids
  }
}

output "platform_addon_contract" {
  value = {
    cluster_addon_profile = local.cluster_addon_profile
    managed_addons = {
      csi_ebs = local.install_ebs_csi
      csi_efs = local.install_efs_csi
    }
    platform_addons = {
      ack_ingress_like_controller = {
        enabled = local.install_lb_controller
        mode    = "future_platform_managed"
      }
      metrics_server = {
        enabled = local.install_metrics_server
        mode    = "future_platform_managed"
      }
    }
  }
}
