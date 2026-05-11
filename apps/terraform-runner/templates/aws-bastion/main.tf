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
  region        = try(var.input.region, "ap-southeast-1")
  environment   = try(var.input.environment, "dev")
  bastion_name  = try(var.input.bastion_name, format("%s-%s", var.blueprint_code, local.environment))
  network_ref   = try(var.input.network_ref, "")
  foundation_stack_name = try(var.input.foundation_stack_name, "")
  name_prefix   = try(var.input.name_prefix, "")
  ops_refs      = try(var.input.ops_refs, [])
  provider_network_refs = try(var.input.provider_network_refs, {})
  vpc_id        = try(var.input.vpc_id, "")
  subnet_id     = try(var.input.subnet_id, "")
  instance_type = try(var.input.instance_type, "t3.micro")
  ssh_key_name  = try(var.input.key_pair_name, "")
  allocate_eip  = try(var.input.allocate_eip, true)
  ingress_cidrs = try(var.input.ingress_cidrs, ["0.0.0.0/0"])
  default_tags  = tomap(try(var.input.default_tags, {}))
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
    provider       = var.cloud_provider
    account_id     = var.account_id
    blueprint_code = var.blueprint_code
    network_plan_id = var.network_plan_id
    action         = var.action
    region         = local.region
    network_ref    = local.network_ref
    foundation_stack_name = local.foundation_stack_name
    name_prefix    = local.name_prefix
    ops_refs       = local.ops_refs
    provider_network_refs = local.provider_network_refs
    bastion_name   = local.bastion_name
    instance_type  = local.instance_type
    vpc_id         = local.vpc_id
    subnet_id      = local.subnet_id
  }
}

provider "aws" {
  region                      = local.region
  skip_credentials_validation = true
  skip_metadata_api_check     = true
  skip_requesting_account_id  = true
  skip_region_validation      = true
}

data "aws_ami" "amazon_linux" {
  most_recent = true
  owners      = ["amazon"]

  filter {
    name   = "name"
    values = ["al2023-ami-2023.*-x86_64"]
  }
}

resource "aws_security_group" "bastion" {
  name        = "${replace(local.bastion_name, "_", "-")}-sg"
  description = "Bastion access security group"
  vpc_id      = local.vpc_id

  dynamic "ingress" {
    for_each = local.ingress_cidrs
    content {
      description = "SSH"
      from_port   = 22
      to_port     = 22
      protocol    = "tcp"
      cidr_blocks = [ingress.value]
    }
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = merge(local.common_tags, {
    Name = "${local.bastion_name}-sg"
  })
}

resource "aws_iam_role" "bastion" {
  name = "${replace(local.bastion_name, "_", "-")}-role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Principal = {
          Service = "ec2.amazonaws.com"
        }
        Action = "sts:AssumeRole"
      }
    ]
  })

  tags = merge(local.common_tags, {
    Name = "${local.bastion_name}-role"
  })
}

resource "aws_iam_role_policy_attachment" "ssm" {
  role       = aws_iam_role.bastion.name
  policy_arn = "arn:aws:iam::aws:policy/AmazonSSMManagedInstanceCore"
}

resource "aws_iam_instance_profile" "bastion" {
  name = "${replace(local.bastion_name, "_", "-")}-profile"
  role = aws_iam_role.bastion.name

  tags = merge(local.common_tags, {
    Name = "${local.bastion_name}-profile"
  })
}

resource "aws_instance" "bastion" {
  ami                         = data.aws_ami.amazon_linux.id
  instance_type               = local.instance_type
  subnet_id                   = local.subnet_id
  vpc_security_group_ids      = [aws_security_group.bastion.id]
  key_name                    = local.ssh_key_name != "" ? local.ssh_key_name : null
  associate_public_ip_address = local.allocate_eip
  iam_instance_profile        = aws_iam_instance_profile.bastion.name

  metadata_options {
    http_tokens = "required"
  }

  tags = merge(local.common_tags, {
    Name = local.bastion_name
    Role = "bastion"
  })
}

resource "aws_eip" "bastion" {
  count    = local.allocate_eip ? 1 : 0
  domain   = "vpc"
  instance = aws_instance.bastion.id

  tags = merge(local.common_tags, {
    Name = "${local.bastion_name}-eip"
  })
}

output "plan_context" {
  value = local.request
}

output "instance_id" {
  value = aws_instance.bastion.id
}

output "private_ip" {
  value = aws_instance.bastion.private_ip
}

output "public_ip" {
  value = local.allocate_eip ? aws_eip.bastion[0].public_ip : aws_instance.bastion.public_ip
}

output "ssh_endpoint" {
  value = format("ssh ec2-user@%s", local.allocate_eip ? aws_eip.bastion[0].public_ip : aws_instance.bastion.public_ip)
}

output "access_entry" {
  value = local.ssh_key_name != "" ? format("ssh ec2-user@%s", local.allocate_eip ? aws_eip.bastion[0].public_ip : aws_instance.bastion.public_ip) : format("aws ssm start-session --target %s --region %s", aws_instance.bastion.id, local.region)
}

output "security_group_id" {
  value = aws_security_group.bastion.id
}

output "vpc_id" {
  value = local.vpc_id
}

output "subnet_id" {
  value = local.subnet_id
}

output "instance_profile_name" {
  value = aws_iam_instance_profile.bastion.name
}

output "network_ref" {
  value = local.network_ref
}

output "foundation_stack_name" {
  value = local.foundation_stack_name
}
