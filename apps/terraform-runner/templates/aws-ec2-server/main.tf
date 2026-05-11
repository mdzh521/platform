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
  region                 = try(var.input.region, "ap-southeast-1")
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
  subnet_id              = try(var.input.subnet_id, "")
  subnet_ids             = try(var.input.subnet_ids, [])
  server_instances_input = try(var.input.server_instances, [])
  instance_type          = try(var.input.instance_type, "t3.micro")
  system_disk_size_gb    = try(var.input.system_disk_size_gb, 40)
  system_disk_type       = try(var.input.system_disk_type, "gp3")
  system_disk_iops       = try(var.input.system_disk_iops, 3000)
  system_disk_throughput = try(var.input.system_disk_throughput, 125)
  data_disks_input       = try(var.input.data_disks, [])
  ssh_key_name           = try(var.input.key_pair_name, "")
  allocate_eip           = try(var.input.allocate_eip, true)
  ingress_cidrs          = try(var.input.ingress_cidrs, ["0.0.0.0/0"])
  login_username         = "ec2-user"
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
  resource_name_parts  = regexall("[0-9a-z_=+,.@-]+", lower(local.server_name))
  resource_name        = length(local.resource_name_parts) > 0 ? join("-", local.resource_name_parts) : lower(var.blueprint_code)
  effective_subnet_ids = length(local.subnet_ids) > 0 ? [for item in local.subnet_ids : tostring(item)] : compact([local.subnet_id])
  default_server_instances = length(local.effective_subnet_ids) == 0 ? [] : [
    for index in range(local.server_count) : {
      name      = local.server_count > 1 ? format("%s-%02d", local.server_name, index + 1) : local.server_name
      subnet_id = local.effective_subnet_ids[index % length(local.effective_subnet_ids)]
    }
  ]
  normalized_server_instances = length(local.server_instances_input) > 0 ? [
    for index, item in local.server_instances_input : {
      name      = try(item.name, local.server_count > 1 ? format("%s-%02d", local.server_name, index + 1) : local.server_name)
      subnet_id = try(item.subnet_id, length(local.effective_subnet_ids) > 0 ? local.effective_subnet_ids[index % length(local.effective_subnet_ids)] : local.subnet_id)
    }
  ] : local.default_server_instances
  server_instances = [
    for item in local.normalized_server_instances : {
      name      = tostring(item.name)
      subnet_id = tostring(item.subnet_id)
    }
    if trimspace(tostring(try(item.name, ""))) != "" && trimspace(tostring(try(item.subnet_id, ""))) != ""
  ]
  server_map = {
    for item in local.server_instances : item.name => item
  }
  data_disks = [
    for item in local.data_disks_input : {
      device_name = tostring(try(item.device_name, ""))
      size_gb     = tonumber(try(item.size_gb, 0))
      volume_type = tostring(try(item.volume_type, "gp3"))
      iops        = tonumber(try(item.iops, 0))
      throughput  = tonumber(try(item.throughput, 0))
    }
    if trimspace(tostring(try(item.device_name, ""))) != "" && tonumber(try(item.size_gb, 0)) > 0
  ]
  primary_server_name = try(keys(local.server_map)[0], "")

  request = {
    provider               = var.cloud_provider
    account_id             = var.account_id
    blueprint_code         = var.blueprint_code
    network_plan_id        = var.network_plan_id
    action                 = var.action
    region                 = local.region
    network_ref            = local.network_ref
    foundation_stack_name  = local.foundation_stack_name
    name_prefix            = local.name_prefix
    project_name           = local.project_name
    workload_refs          = local.workload_refs
    ops_refs               = local.ops_refs
    provider_network_refs  = local.provider_network_refs
    server_name            = local.server_name
    server_count           = local.server_count
    instance_type          = local.instance_type
    system_disk_size_gb    = local.system_disk_size_gb
    system_disk_type       = local.system_disk_type
    system_disk_iops       = local.system_disk_iops
    system_disk_throughput = local.system_disk_throughput
    data_disks             = local.data_disks
    vpc_id                 = local.vpc_id
    subnet_id              = local.subnet_id
    subnet_ids             = local.subnet_ids
    server_instances       = local.server_instances
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

resource "aws_security_group" "server" {
  name        = "${local.resource_name}-sg"
  description = "Application server access security group"
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
    Name = "${local.resource_name}-sg"
  })
}

resource "aws_iam_role" "server" {
  name = "${local.resource_name}-role"

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
    Name = "${local.resource_name}-role"
  })
}

resource "aws_iam_role_policy_attachment" "ssm" {
  role       = aws_iam_role.server.name
  policy_arn = "arn:aws:iam::aws:policy/AmazonSSMManagedInstanceCore"
}

resource "aws_iam_instance_profile" "server" {
  name = "${local.resource_name}-profile"
  role = aws_iam_role.server.name

  tags = merge(local.common_tags, {
    Name = "${local.resource_name}-profile"
  })
}

resource "aws_instance" "server" {
  for_each                    = local.server_map
  ami                         = data.aws_ami.amazon_linux.id
  instance_type               = local.instance_type
  subnet_id                   = each.value.subnet_id
  vpc_security_group_ids      = [aws_security_group.server.id]
  key_name                    = local.ssh_key_name != "" ? local.ssh_key_name : null
  associate_public_ip_address = local.allocate_eip
  iam_instance_profile        = aws_iam_instance_profile.server.name

  root_block_device {
    volume_size = local.system_disk_size_gb
    volume_type = local.system_disk_type
    iops        = contains(["gp3", "io1", "io2"], local.system_disk_type) ? local.system_disk_iops : null
    throughput  = contains(["gp3"], local.system_disk_type) ? local.system_disk_throughput : null
    encrypted   = true
    tags = merge(local.common_tags, {
      Name      = "${each.value.name}-root"
      Role      = "server-root-disk"
      ServerRef = each.value.name
    })
  }

  dynamic "ebs_block_device" {
    for_each = {
      for disk in local.data_disks : disk.device_name => disk
    }
    content {
      device_name = ebs_block_device.value.device_name
      volume_size = ebs_block_device.value.size_gb
      volume_type = ebs_block_device.value.volume_type
      iops        = contains(["gp3", "io1", "io2"], ebs_block_device.value.volume_type) ? ebs_block_device.value.iops : null
      throughput  = contains(["gp3"], ebs_block_device.value.volume_type) ? ebs_block_device.value.throughput : null
      encrypted   = true
      tags = merge(local.common_tags, {
        Name      = "${each.value.name}-${replace(ebs_block_device.value.device_name, "/dev/", "")}"
        Role      = "server-data-disk"
        ServerRef = each.value.name
      })
    }
  }

  metadata_options {
    http_tokens = "required"
  }

  tags = merge(local.common_tags, {
    Name = each.value.name
    Role = "server"
  })
}

resource "aws_eip" "server" {
  for_each = local.allocate_eip ? local.server_map : {}
  domain   = "vpc"
  instance = aws_instance.server[each.key].id

  tags = merge(local.common_tags, {
    Name = "${each.key}-eip"
    Role = "server-eip"
  })
}

resource "aws_ec2_tag" "server_primary_eni_name" {
  for_each    = local.server_map
  resource_id = aws_instance.server[each.key].primary_network_interface_id
  key         = "Name"
  value       = "${each.key}-eni"
}

resource "aws_ec2_tag" "server_primary_eni_role" {
  for_each    = local.server_map
  resource_id = aws_instance.server[each.key].primary_network_interface_id
  key         = "Role"
  value       = "server-primary-eni"
}

resource "aws_ec2_tag" "server_primary_eni_managed" {
  for_each    = local.server_map
  resource_id = aws_instance.server[each.key].primary_network_interface_id
  key         = "ManagedBy"
  value       = "platform-center"
}

resource "aws_ec2_tag" "server_primary_eni_blueprint" {
  for_each    = local.server_map
  resource_id = aws_instance.server[each.key].primary_network_interface_id
  key         = "BlueprintCode"
  value       = var.blueprint_code
}

resource "aws_ec2_tag" "server_primary_eni_environment" {
  for_each    = local.server_map
  resource_id = aws_instance.server[each.key].primary_network_interface_id
  key         = "Environment"
  value       = local.environment
}

output "plan_context" {
  value = local.request
}

output "instance_id" {
  value = try(aws_instance.server[local.primary_server_name].id, "")
}

output "private_ip" {
  value = try(aws_instance.server[local.primary_server_name].private_ip, "")
}

output "public_ip" {
  value = local.allocate_eip ? try(aws_eip.server[local.primary_server_name].public_ip, "") : try(aws_instance.server[local.primary_server_name].public_ip, "")
}

output "ssh_endpoint" {
  value = format("ssh %s@%s", local.login_username, local.allocate_eip ? try(aws_eip.server[local.primary_server_name].public_ip, "") : try(aws_instance.server[local.primary_server_name].public_ip, ""))
}

output "access_entry" {
  value = local.ssh_key_name != "" ? format("ssh %s@%s", local.login_username, local.allocate_eip ? try(aws_eip.server[local.primary_server_name].public_ip, "") : try(aws_instance.server[local.primary_server_name].public_ip, "")) : format("aws ssm start-session --target %s --region %s", try(aws_instance.server[local.primary_server_name].id, ""), local.region)
}

output "security_group_id" {
  value = aws_security_group.server.id
}

output "vpc_id" {
  value = local.vpc_id
}

output "subnet_id" {
  value = local.subnet_id
}

output "instance_profile_name" {
  value = aws_iam_instance_profile.server.name
}

output "network_ref" {
  value = local.network_ref
}

output "foundation_stack_name" {
  value = local.foundation_stack_name
}

output "instance_ids" {
  value = {
    for name, item in aws_instance.server : name => item.id
  }
}

output "private_ips" {
  value = {
    for name, item in aws_instance.server : name => item.private_ip
  }
}

output "public_ips" {
  value = local.allocate_eip ? {
    for name, item in aws_eip.server : name => item.public_ip
    } : {
    for name, item in aws_instance.server : name => item.public_ip
  }
}

output "server_instances" {
  value = [
    for name, item in aws_instance.server : {
      name        = name
      instance_id = item.id
      subnet_id   = item.subnet_id
      private_ip  = item.private_ip
      public_ip   = local.allocate_eip ? try(aws_eip.server[name].public_ip, "") : item.public_ip
    }
  ]
}

output "access_entries" {
  value = [
    for name, item in aws_instance.server : {
      name         = name
      login_user   = local.login_username
      instance_id  = item.id
      public_ip    = local.allocate_eip ? try(aws_eip.server[name].public_ip, "") : item.public_ip
      private_ip   = item.private_ip
      access_entry = local.ssh_key_name != "" ? format("ssh %s@%s", local.login_username, local.allocate_eip ? try(aws_eip.server[name].public_ip, "") : item.public_ip) : format("aws ssm start-session --target %s --region %s", item.id, local.region)
    }
  ]
}

output "login_username" {
  value = local.login_username
}
