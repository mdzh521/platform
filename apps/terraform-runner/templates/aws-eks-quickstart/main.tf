terraform {
  required_version = ">= 1.6.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
    tls = {
      source  = "hashicorp/tls"
      version = "~> 4.0"
    }
  }
}

locals {
  region                = try(var.input.region, "ap-southeast-1")
  environment           = try(var.input.environment, "dev")
  cluster_name          = try(var.input.cluster_name, "demo-eks")
  kubernetes_version    = try(var.input.kubernetes_version, "1.31")
  node_instance_type    = try(var.input.node_instance_type, "t3.large")
  desired_capacity      = tonumber(try(var.input.desired_capacity, 2))
  min_size              = tonumber(try(var.input.min_size, 2))
  max_size              = tonumber(try(var.input.max_size, 4))
  network_ref           = try(var.input.network_ref, "")
  foundation_stack_name = try(var.input.foundation_stack_name, "")
  cluster_node_refs     = try(var.input.cluster_node_refs, [])
  pod_network_refs      = try(var.input.pod_network_refs, [])
  provider_network_refs = try(var.input.provider_network_refs, {})
  ops_refs              = try(var.input.ops_refs, [])
  endpoint_access       = lower(try(var.input.endpoint_access, "private"))
  vpc_id                = try(var.input.vpc_id, "")
  subnet_ids            = distinct(compact([for item in try(var.input.subnet_ids, []) : tostring(item)]))
  default_tags          = tomap(try(var.input.default_tags, {}))
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

  endpoint_public_access  = local.endpoint_access == "public" || local.endpoint_access == "public_and_private"
  endpoint_private_access = local.endpoint_access != "public"
  cluster_role_name       = "${local.cluster_name}-cluster-role"
  node_role_name          = "${local.cluster_name}-node-role"
  cluster_sg_name         = "${local.cluster_name}-cluster-sg"
  node_sg_name            = "${local.cluster_name}-node-sg"
  node_group_name         = "${local.cluster_name}-managed-ng"
  cluster_addon_profile   = lower(try(var.input.cluster_addon_profile, "standard"))
  addon_profile_defaults = {
    minimal = {
      install_ebs_csi        = false
      install_efs_csi        = false
      install_lb_controller  = false
      install_metrics_server = false
    }
    standard = {
      install_ebs_csi        = true
      install_efs_csi        = true
      install_lb_controller  = true
      install_metrics_server = false
    }
    platform = {
      install_ebs_csi        = true
      install_efs_csi        = true
      install_lb_controller  = true
      install_metrics_server = false
    }
  }
  selected_addon_profile = lookup(local.addon_profile_defaults, local.cluster_addon_profile, local.addon_profile_defaults.standard)
  install_ebs_csi        = try(tobool(var.input.install_ebs_csi), local.selected_addon_profile.install_ebs_csi)
  install_efs_csi        = try(tobool(var.input.install_efs_csi), local.selected_addon_profile.install_efs_csi)
  install_lb_controller  = try(tobool(var.input.install_lb_controller), local.selected_addon_profile.install_lb_controller)
  install_metrics_server = try(tobool(var.input.install_metrics_server), local.selected_addon_profile.install_metrics_server)
  manage_eks_addons      = true
  enable_irsa_roles      = local.install_ebs_csi || local.install_efs_csi || local.install_lb_controller
  base_eks_addons = {
    coredns    = {}
    kube-proxy = {}
    vpc-cni    = {}
  }
  oidc_issuer_url             = aws_eks_cluster.this.identity[0].oidc[0].issuer
  oidc_provider_host          = trimprefix(local.oidc_issuer_url, "https://")
  ebs_csi_role_name           = "${local.cluster_name}-ebs-csi-role"
  efs_csi_role_name           = "${local.cluster_name}-efs-csi-role"
  load_balancer_role_name     = "${local.cluster_name}-aws-load-balancer-controller-role"
  load_balancer_policy_name   = "${local.cluster_name}-AWSLoadBalancerControllerIAMPolicy"
  load_balancer_chart_name    = "aws-load-balancer-controller"
  load_balancer_chart_repo    = "https://aws.github.io/eks-charts"
  load_balancer_chart_version = "1.14.0"

  request = {
    provider               = var.cloud_provider
    account_id             = var.account_id
    blueprint_code         = var.blueprint_code
    network_plan_id        = var.network_plan_id
    action                 = var.action
    region                 = local.region
    environment            = local.environment
    cluster_name           = local.cluster_name
    kubernetes_version     = local.kubernetes_version
    node_instance_type     = local.node_instance_type
    desired_capacity       = local.desired_capacity
    min_size               = local.min_size
    max_size               = local.max_size
    network_ref            = local.network_ref
    foundation_stack_name  = local.foundation_stack_name
    cluster_node_refs      = local.cluster_node_refs
    pod_network_refs       = local.pod_network_refs
    provider_network_refs  = local.provider_network_refs
    ops_refs               = local.ops_refs
    endpoint_access        = local.endpoint_access
    vpc_id                 = local.vpc_id
    subnet_ids             = local.subnet_ids
    cluster_addon_profile  = local.cluster_addon_profile
    install_ebs_csi        = local.install_ebs_csi
    install_efs_csi        = local.install_efs_csi
    install_lb_controller  = local.install_lb_controller
    install_metrics_server = local.install_metrics_server
  }
}

provider "aws" {
  region                      = local.region
  skip_credentials_validation = true
  skip_metadata_api_check     = true
  skip_requesting_account_id  = true
  skip_region_validation      = true
}

data "aws_iam_policy_document" "cluster_assume_role" {
  statement {
    effect = "Allow"

    principals {
      type        = "Service"
      identifiers = ["eks.amazonaws.com"]
    }

    actions = ["sts:AssumeRole"]
  }
}

data "aws_iam_policy_document" "node_assume_role" {
  statement {
    effect = "Allow"

    principals {
      type        = "Service"
      identifiers = ["ec2.amazonaws.com"]
    }

    actions = ["sts:AssumeRole"]
  }
}

data "tls_certificate" "cluster_oidc" {
  count = local.enable_irsa_roles ? 1 : 0
  url   = local.oidc_issuer_url
}

resource "aws_iam_openid_connect_provider" "cluster" {
  count           = local.enable_irsa_roles ? 1 : 0
  url             = local.oidc_issuer_url
  client_id_list  = ["sts.amazonaws.com"]
  thumbprint_list = [data.tls_certificate.cluster_oidc[0].certificates[0].sha1_fingerprint]

  tags = merge(local.common_tags, {
    Name = "${local.cluster_name}-oidc"
    Role = "eks-oidc-provider"
  })
}

data "aws_iam_policy_document" "ebs_csi_assume_role" {
  count = local.install_ebs_csi ? 1 : 0

  statement {
    effect  = "Allow"
    actions = ["sts:AssumeRoleWithWebIdentity"]

    principals {
      type        = "Federated"
      identifiers = [aws_iam_openid_connect_provider.cluster[0].arn]
    }

    condition {
      test     = "StringEquals"
      variable = "${local.oidc_provider_host}:aud"
      values   = ["sts.amazonaws.com"]
    }

    condition {
      test     = "StringEquals"
      variable = "${local.oidc_provider_host}:sub"
      values   = ["system:serviceaccount:kube-system:ebs-csi-controller-sa"]
    }
  }
}

data "aws_iam_policy_document" "efs_csi_assume_role" {
  count = local.install_efs_csi ? 1 : 0

  statement {
    effect  = "Allow"
    actions = ["sts:AssumeRoleWithWebIdentity"]

    principals {
      type        = "Federated"
      identifiers = [aws_iam_openid_connect_provider.cluster[0].arn]
    }

    condition {
      test     = "StringEquals"
      variable = "${local.oidc_provider_host}:aud"
      values   = ["sts.amazonaws.com"]
    }

    condition {
      test     = "StringLike"
      variable = "${local.oidc_provider_host}:sub"
      values   = ["system:serviceaccount:kube-system:efs-csi-*"]
    }
  }
}

data "aws_iam_policy_document" "load_balancer_controller_assume_role" {
  count = local.install_lb_controller ? 1 : 0

  statement {
    effect  = "Allow"
    actions = ["sts:AssumeRoleWithWebIdentity"]

    principals {
      type        = "Federated"
      identifiers = [aws_iam_openid_connect_provider.cluster[0].arn]
    }

    condition {
      test     = "StringEquals"
      variable = "${local.oidc_provider_host}:aud"
      values   = ["sts.amazonaws.com"]
    }

    condition {
      test     = "StringEquals"
      variable = "${local.oidc_provider_host}:sub"
      values   = ["system:serviceaccount:kube-system:aws-load-balancer-controller"]
    }
  }
}

resource "aws_iam_role" "cluster" {
  name               = local.cluster_role_name
  assume_role_policy = data.aws_iam_policy_document.cluster_assume_role.json

  tags = merge(local.common_tags, {
    Name = local.cluster_role_name
    Role = "eks-cluster-role"
  })
}

resource "aws_iam_role_policy_attachment" "cluster" {
  for_each = toset([
    "arn:aws:iam::aws:policy/AmazonEKSClusterPolicy",
    "arn:aws:iam::aws:policy/AmazonEKSVPCResourceController",
  ])

  role       = aws_iam_role.cluster.name
  policy_arn = each.value
}

resource "aws_iam_role" "node" {
  name               = local.node_role_name
  assume_role_policy = data.aws_iam_policy_document.node_assume_role.json

  tags = merge(local.common_tags, {
    Name = local.node_role_name
    Role = "eks-node-role"
  })
}

resource "aws_iam_role_policy_attachment" "node" {
  for_each = toset([
    "arn:aws:iam::aws:policy/AmazonEKSWorkerNodePolicy",
    "arn:aws:iam::aws:policy/AmazonEKS_CNI_Policy",
    "arn:aws:iam::aws:policy/AmazonEC2ContainerRegistryPullOnly",
    "arn:aws:iam::aws:policy/AmazonSSMManagedInstanceCore",
  ])

  role       = aws_iam_role.node.name
  policy_arn = each.value
}

resource "aws_security_group" "cluster" {
  name        = local.cluster_sg_name
  description = "EKS control plane security group"
  vpc_id      = local.vpc_id

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = merge(local.common_tags, {
    Name = local.cluster_sg_name
    Role = "eks-cluster-sg"
  })
}

resource "aws_security_group" "node" {
  name        = local.node_sg_name
  description = "EKS node security group"
  vpc_id      = local.vpc_id

  ingress {
    description     = "Allow cluster API to kubelet"
    from_port       = 1025
    to_port         = 65535
    protocol        = "tcp"
    security_groups = [aws_security_group.cluster.id]
  }

  ingress {
    description = "Node to node"
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    self        = true
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = merge(local.common_tags, {
    Name = local.node_sg_name
    Role = "eks-node-sg"
  })
}

resource "aws_eks_cluster" "this" {
  name                          = local.cluster_name
  role_arn                      = aws_iam_role.cluster.arn
  version                       = local.kubernetes_version
  bootstrap_self_managed_addons = false

  vpc_config {
    subnet_ids              = local.subnet_ids
    security_group_ids      = [aws_security_group.cluster.id]
    endpoint_private_access = local.endpoint_private_access
    endpoint_public_access  = local.endpoint_public_access
  }

  depends_on = [
    aws_iam_role_policy_attachment.cluster,
  ]

  tags = merge(local.common_tags, {
    Name = local.cluster_name
    Role = "eks-cluster"
  })
}

resource "aws_eks_node_group" "default" {
  cluster_name    = aws_eks_cluster.this.name
  node_group_name = local.node_group_name
  node_role_arn   = aws_iam_role.node.arn
  subnet_ids      = local.subnet_ids
  instance_types  = [local.node_instance_type]
  version         = local.kubernetes_version

  scaling_config {
    desired_size = local.desired_capacity
    min_size     = local.min_size
    max_size     = local.max_size
  }

  update_config {
    max_unavailable = 1
  }

  depends_on = [
    aws_iam_role_policy_attachment.node,
    aws_eks_cluster.this,
  ]

  tags = merge(local.common_tags, {
    Name = local.node_group_name
    Role = "eks-node-group"
  })
}

resource "aws_iam_role" "ebs_csi" {
  count              = local.install_ebs_csi ? 1 : 0
  name               = local.ebs_csi_role_name
  assume_role_policy = data.aws_iam_policy_document.ebs_csi_assume_role[0].json

  tags = merge(local.common_tags, {
    Name = local.ebs_csi_role_name
    Role = "eks-addon-ebs-csi"
  })
}

resource "aws_iam_role_policy_attachment" "ebs_csi" {
  count      = local.install_ebs_csi ? 1 : 0
  role       = aws_iam_role.ebs_csi[0].name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonEBSCSIDriverPolicy"
}

resource "aws_iam_role" "efs_csi" {
  count              = local.install_efs_csi ? 1 : 0
  name               = local.efs_csi_role_name
  assume_role_policy = data.aws_iam_policy_document.efs_csi_assume_role[0].json

  tags = merge(local.common_tags, {
    Name = local.efs_csi_role_name
    Role = "eks-addon-efs-csi"
  })
}

resource "aws_iam_role_policy_attachment" "efs_csi" {
  count      = local.install_efs_csi ? 1 : 0
  role       = aws_iam_role.efs_csi[0].name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonEFSCSIDriverPolicy"
}

resource "aws_iam_policy" "load_balancer_controller" {
  count  = local.install_lb_controller ? 1 : 0
  name   = local.load_balancer_policy_name
  policy = file("${path.module}/aws-load-balancer-controller-iam-policy.json")

  tags = merge(local.common_tags, {
    Name = local.load_balancer_policy_name
    Role = "eks-addon-lb-controller-policy"
  })
}

resource "aws_iam_role" "load_balancer_controller" {
  count              = local.install_lb_controller ? 1 : 0
  name               = local.load_balancer_role_name
  assume_role_policy = data.aws_iam_policy_document.load_balancer_controller_assume_role[0].json

  tags = merge(local.common_tags, {
    Name = local.load_balancer_role_name
    Role = "eks-addon-lb-controller"
  })
}

resource "aws_iam_role_policy_attachment" "load_balancer_controller" {
  count      = local.install_lb_controller ? 1 : 0
  role       = aws_iam_role.load_balancer_controller[0].name
  policy_arn = aws_iam_policy.load_balancer_controller[0].arn
}

resource "aws_eks_addon" "base" {
  for_each = local.manage_eks_addons ? local.base_eks_addons : {}

  cluster_name                = aws_eks_cluster.this.name
  addon_name                  = each.key
  resolve_conflicts_on_create = "OVERWRITE"
  resolve_conflicts_on_update = "OVERWRITE"
  tags                        = local.common_tags

  depends_on = [
    aws_eks_node_group.default,
  ]
}

resource "aws_eks_addon" "ebs_csi" {
  count = local.install_ebs_csi ? 1 : 0

  cluster_name                = aws_eks_cluster.this.name
  addon_name                  = "aws-ebs-csi-driver"
  service_account_role_arn    = aws_iam_role.ebs_csi[0].arn
  resolve_conflicts_on_create = "OVERWRITE"
  resolve_conflicts_on_update = "OVERWRITE"
  tags                        = local.common_tags

  depends_on = [
    aws_eks_node_group.default,
    aws_iam_role_policy_attachment.ebs_csi,
  ]
}

resource "aws_eks_addon" "efs_csi" {
  count = local.install_efs_csi ? 1 : 0

  cluster_name                = aws_eks_cluster.this.name
  addon_name                  = "aws-efs-csi-driver"
  service_account_role_arn    = aws_iam_role.efs_csi[0].arn
  resolve_conflicts_on_create = "OVERWRITE"
  resolve_conflicts_on_update = "OVERWRITE"
  tags                        = local.common_tags

  depends_on = [
    aws_eks_node_group.default,
    aws_iam_role_policy_attachment.efs_csi,
  ]
}

output "plan_context" {
  value = local.request
}

output "cluster_name" {
  value = aws_eks_cluster.this.name
}

output "cluster_arn" {
  value = aws_eks_cluster.this.arn
}

output "cluster_endpoint" {
  value = aws_eks_cluster.this.endpoint
}

output "cluster_status" {
  value = aws_eks_cluster.this.status
}

output "cluster_version" {
  value = aws_eks_cluster.this.version
}

output "node_group_name" {
  value = aws_eks_node_group.default.node_group_name
}

output "vpc_id" {
  value = local.vpc_id
}

output "subnet_ids" {
  value = local.subnet_ids
}

output "cluster_addon_profile" {
  value = local.cluster_addon_profile
}

output "managed_addons" {
  value = {
    coredns            = try(aws_eks_addon.base["coredns"].addon_name, null)
    kube_proxy         = try(aws_eks_addon.base["kube-proxy"].addon_name, null)
    vpc_cni            = try(aws_eks_addon.base["vpc-cni"].addon_name, null)
    aws_ebs_csi_driver = try(aws_eks_addon.ebs_csi[0].addon_name, null)
    aws_efs_csi_driver = try(aws_eks_addon.efs_csi[0].addon_name, null)
  }
}

output "platform_addon_contract" {
  value = {
    cluster_addon_profile = local.cluster_addon_profile
    managed_addons = {
      coredns            = true
      kube_proxy         = true
      vpc_cni            = true
      aws_ebs_csi_driver = local.install_ebs_csi
      aws_efs_csi_driver = local.install_efs_csi
    }
    helm_addons = {
      aws_load_balancer_controller = {
        enabled              = local.install_lb_controller
        install_mode         = "platform_managed_helm"
        chart_repository     = local.load_balancer_chart_repo
        chart_name           = local.load_balancer_chart_name
        chart_version        = local.load_balancer_chart_version
        namespace            = "kube-system"
        service_account_name = "aws-load-balancer-controller"
        iam_role_arn         = try(aws_iam_role.load_balancer_controller[0].arn, "")
        iam_policy_arn       = try(aws_iam_policy.load_balancer_controller[0].arn, "")
        cluster_name         = aws_eks_cluster.this.name
        region               = local.region
        vpc_id               = local.vpc_id
        endpoint_access      = local.endpoint_access
        notes                = "Current runtime keeps this as a platform-managed Helm addon contract; private-only API endpoints are not installed directly from terraform-runner."
      }
    }
    recommended_future_addons = {
      metrics_server = {
        enabled      = local.install_metrics_server
        install_mode = "future_contract"
        notes        = "Phase6 keeps metrics-server out of the standard cluster baseline until the platform-managed Helm addon execution chain is validated."
      }
    }
  }
}
