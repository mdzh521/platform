package blueprints

import (
	"backend-center/internal/domains/delivery/cloud"

	"gorm.io/gorm"
)

type BlueprintService struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) *BlueprintService {
	return &BlueprintService{db: db}
}

func (s *BlueprintService) List() ([]cloud.BlueprintView, error) {
	var items []cloud.DeploymentBlueprint
	err := s.db.Where("enabled = ?", true).Order("provider asc, category asc, id asc").Find(&items).Error
	if err != nil {
		return nil, err
	}
	result := make([]cloud.BlueprintView, 0, len(items))
	for _, item := range items {
		result = append(result, cloud.BlueprintView{
			ID:              item.ID,
			Code:            item.Code,
			Name:            item.Name,
			Provider:        item.Provider,
			Category:        item.Category,
			Version:         item.Version,
			TemplatePath:    item.TemplatePath,
			SchemaJSON:      item.SchemaJSON,
			Description:     item.Description,
			Maturity:        item.Maturity,
			Capability:      item.Capability,
			SupportsApply:   cloud.BlueprintSupportsApply(item.Capability),
			SupportsDestroy: cloud.BlueprintSupportsDestroy(item.Capability),
			Enabled:         item.Enabled,
			CreatedAt:       item.CreatedAt,
			UpdatedAt:       item.UpdatedAt,
		})
	}
	return result, nil
}

func (s *BlueprintService) EnsureDefaultBlueprints() {
	defaults := []cloud.DeploymentBlueprint{
		{Code: "aws-vpc-base", Name: "AWS 基础网络", Provider: "aws", Category: "network", Version: "v1", TemplatePath: "terraform-runner/templates/aws-vpc-base", SchemaJSON: `{"required":["region","vpc_cidr","availability_zone_count","subnet_groups"],"properties":{"region":{"type":"string"},"environment":{"type":"string"},"vpc_name":{"type":"string"},"vpc_cidr":{"type":"string"},"availability_zone_count":{"type":"number"},"subnet_groups":{"type":"array","items":{"type":"object","properties":{"role":{"type":"string"},"tier":{"type":"string"},"cidrs":{"type":"array","items":{"type":"string"}},"description":{"type":"string"}}}},"nat_gateway_count":{"type":"number"},"create_bastion_subnet":{"type":"boolean"},"bastion_subnet_cidr":{"type":"string"},"default_tags":{"type":"object"}}}`, Description: "创建标准 AWS 基础网络，收口多角色 subnet、路由、NAT 与可复用输出。", Maturity: "apply ready", Capability: "apply_destroy_ready", Enabled: true},
		{Code: "aws-eks-quickstart", Name: "AWS Cluster Blueprint", Provider: "aws", Category: "cluster", Version: "v1", TemplatePath: "terraform-runner/templates/aws-eks-quickstart", SchemaJSON: `{"required":["region","cluster_name","node_instance_type","network_ref","foundation_stack_name","cluster_node_refs","pod_network_refs","provider_network_refs","ops_refs"],"properties":{"region":{"type":"string"},"environment":{"type":"string"},"cluster_name":{"type":"string"},"kubernetes_version":{"type":"string"},"node_instance_type":{"type":"string"},"desired_capacity":{"type":"number"},"min_size":{"type":"number"},"max_size":{"type":"number"},"network_ref":{"type":"string"},"foundation_stack_name":{"type":"string"},"cluster_node_refs":{"type":"array"},"pod_network_refs":{"type":"array"},"provider_network_refs":{"type":"object"},"ops_refs":{"type":"array"},"endpoint_access":{"type":"string"},"vpc_id":{"type":"string"},"subnet_ids":{"type":"array","items":{"type":"string"}},"cluster_addon_profile":{"type":"string","enum":["minimal","standard","platform"]},"install_ebs_csi":{"type":"boolean"},"install_efs_csi":{"type":"boolean"},"install_lb_controller":{"type":"boolean"},"install_metrics_server":{"type":"boolean"},"default_tags":{"type":"object"}}}`, Description: "基于 AWS 基础网络 直接创建真实 EKS 集群，统一带入 cluster node、pod network、ops 与 provider-specific refs，并表达 EKS managed add-ons 与平台 Helm addon 基线。", Maturity: "experimental", Capability: "experimental_apply", Enabled: true},
		{Code: "aws-bastion", Name: "AWS Ops Bastion", Provider: "aws", Category: "compute", Version: "v1", TemplatePath: "terraform-runner/templates/aws-bastion", SchemaJSON: `{"required":["region","instance_type","network_ref","vpc_id","subnet_id"],"properties":{"region":{"type":"string"},"environment":{"type":"string"},"bastion_name":{"type":"string"},"instance_type":{"type":"string"},"key_pair_name":{"type":"string"},"network_ref":{"type":"string"},"foundation_stack_name":{"type":"string"},"name_prefix":{"type":"string"},"ops_refs":{"type":"array"},"provider_network_refs":{"type":"object"},"vpc_id":{"type":"string"},"subnet_id":{"type":"string"},"allocate_eip":{"type":"boolean"},"ingress_cidrs":{"type":"array","items":{"type":"string"}},"default_tags":{"type":"object"}}}`, Description: "复用已有 基础网络，在指定 VPC / 子网中创建 ops bastion，并接收 基础网络 refs 作为上层网络契约。", Maturity: "experimental", Capability: "apply_destroy_ready", Enabled: true},
		{Code: "aws-ec2-server", Name: "AWS Server", Provider: "aws", Category: "compute", Version: "v1", TemplatePath: "terraform-runner/templates/aws-ec2-server", SchemaJSON: `{"required":["region","instance_type","network_ref","vpc_id"],"properties":{"region":{"type":"string"},"environment":{"type":"string"},"project_name":{"type":"string"},"server_name":{"type":"string"},"server_count":{"type":"number"},"instance_type":{"type":"string"},"system_disk_size_gb":{"type":"number"},"system_disk_type":{"type":"string"},"system_disk_iops":{"type":"number"},"system_disk_throughput":{"type":"number"},"data_disks":{"type":"array","items":{"type":"object","properties":{"device_name":{"type":"string"},"size_gb":{"type":"number"},"volume_type":{"type":"string"},"iops":{"type":"number"},"throughput":{"type":"number"}}}},"key_pair_name":{"type":"string"},"network_ref":{"type":"string"},"foundation_stack_name":{"type":"string"},"name_prefix":{"type":"string"},"workload_refs":{"type":"array"},"ops_refs":{"type":"array"},"provider_network_refs":{"type":"object"},"vpc_id":{"type":"string"},"subnet_id":{"type":"string"},"subnet_ids":{"type":"array","items":{"type":"string"}},"server_instances":{"type":"array","items":{"type":"object"}},"allocate_eip":{"type":"boolean"},"ingress_cidrs":{"type":"array","items":{"type":"string"}},"default_tags":{"type":"object"}}}`, Description: "复用已有 AWS 基础网络，在指定 VPC / 子网中创建一台或多台通用 EC2 Server，并在资源同步后自动注册到机器管理。支持系统盘 / 多数据盘、IOPS、吞吐、批量主机名和主机标签。", Maturity: "experimental", Capability: "apply_destroy_ready", Enabled: true},
		{Code: "alicloud-vpc-base", Name: "阿里云 基础网络", Provider: "alicloud", Category: "network", Version: "v1", TemplatePath: "terraform-runner/templates/alicloud-vpc-base", SchemaJSON: `{"required":["region","vpc_cidr","availability_zone_count","subnet_groups"],"properties":{"region":{"type":"string"},"environment":{"type":"string"},"vpc_name":{"type":"string"},"vpc_cidr":{"type":"string"},"availability_zone_count":{"type":"number"},"subnet_groups":{"type":"array","items":{"type":"object","properties":{"role":{"type":"string"},"traffic_profile":{"type":"string"},"cidrs":{"type":"array","items":{"type":"string"}},"description":{"type":"string"}}}},"nat_gateway_count":{"type":"number"},"create_bastion_subnet":{"type":"boolean"},"bastion_subnet_cidr":{"type":"string"},"default_tags":{"type":"object"}}}`, Description: "创建标准阿里云 基础网络，使用 role-based vSwitch 与 traffic_profile 表达入口、业务、数据、运维和 ACK 网络语义。", Maturity: "apply ready", Capability: "apply_destroy_ready", Enabled: true},
		{Code: "alicloud-ecs-server", Name: "阿里云 Server", Provider: "alicloud", Category: "compute", Version: "v1", TemplatePath: "terraform-runner/templates/alicloud-ecs-server", SchemaJSON: `{"required":["region","instance_type","network_ref","vpc_id"],"properties":{"region":{"type":"string"},"environment":{"type":"string"},"project_name":{"type":"string"},"server_name":{"type":"string"},"server_count":{"type":"number"},"instance_type":{"type":"string"},"instance_charge_type":{"type":"string","enum":["PostPaid","PrePaid"]},"period":{"type":"number"},"period_unit":{"type":"string","enum":["Month"]},"system_disk_size_gb":{"type":"number"},"system_disk_type":{"type":"string"},"key_pair_name":{"type":"string"},"network_ref":{"type":"string"},"foundation_stack_name":{"type":"string"},"name_prefix":{"type":"string"},"workload_refs":{"type":"array"},"ops_refs":{"type":"array"},"provider_network_refs":{"type":"object"},"vpc_id":{"type":"string"},"vswitch_id":{"type":"string"},"subnet_id":{"type":"string"},"subnet_ids":{"type":"array","items":{"type":"string"}},"server_instances":{"type":"array","items":{"type":"object"}},"allocate_eip":{"type":"boolean"},"ingress_cidrs":{"type":"array","items":{"type":"string"}},"default_tags":{"type":"object"}}}`, Description: "复用已有阿里云 基础网络，在指定 VPC / 交换机中创建一台或多台通用 ECS Server，并在资源同步后自动注册到机器管理。第一版先收口基础 ECS、EIP、系统盘、批量主机名与主机标签。", Maturity: "experimental", Capability: "apply_destroy_ready", Enabled: true},
		{Code: "alicloud-ack-quickstart", Name: "阿里云 Cluster Blueprint", Provider: "alicloud", Category: "cluster", Version: "v1", TemplatePath: "terraform-runner/templates/alicloud-ack-quickstart", SchemaJSON: `{"required":["region","cluster_name","worker_instance_type","network_ref","foundation_stack_name","cluster_node_refs","pod_network_refs","provider_network_refs","ops_refs"],"properties":{"region":{"type":"string"},"environment":{"type":"string"},"cluster_name":{"type":"string"},"kubernetes_version":{"type":"string"},"cluster_spec":{"type":"string"},"worker_instance_type":{"type":"string"},"desired_capacity":{"type":"number"},"min_size":{"type":"number"},"max_size":{"type":"number"},"node_system_disk_category":{"type":"string"},"node_system_disk_size":{"type":"number"},"node_login_password":{"type":"string"},"key_pair_name":{"type":"string"},"network_ref":{"type":"string"},"foundation_stack_name":{"type":"string"},"cluster_node_refs":{"type":"array"},"pod_network_refs":{"type":"array"},"provider_network_refs":{"type":"object"},"ops_refs":{"type":"array"},"endpoint_access":{"type":"string"},"vpc_id":{"type":"string"},"subnet_ids":{"type":"array","items":{"type":"string"}},"pod_vswitch_ids":{"type":"array","items":{"type":"string"}},"slb_vswitch_ids":{"type":"array","items":{"type":"string"}},"service_cidr":{"type":"string"},"pod_cidr":{"type":"string"},"create_ack_service_roles":{"type":"boolean"},"cluster_addon_profile":{"type":"string","enum":["minimal","standard","platform"]},"install_ebs_csi":{"type":"boolean"},"install_efs_csi":{"type":"boolean"},"install_lb_controller":{"type":"boolean"},"install_metrics_server":{"type":"boolean"},"default_tags":{"type":"object"}}}`, Description: "基于阿里云 基础网络 直接创建真实 ACK 集群，统一带入 ACK node、pod network、ops 与 provider-specific refs，并预留平台基线插件 contract。", Maturity: "experimental", Capability: "experimental_apply", Enabled: true},
		{Code: "alicloud-bastion", Name: "阿里云 Ops Bastion", Provider: "alicloud", Category: "compute", Version: "v1", TemplatePath: "terraform-runner/templates/alicloud-bastion", SchemaJSON: `{"required":["region","instance_type","network_ref","foundation_stack_name","ops_refs","provider_network_refs"],"properties":{"region":{"type":"string"},"environment":{"type":"string"},"bastion_name":{"type":"string"},"instance_type":{"type":"string"},"instance_charge_type":{"type":"string","enum":["PostPaid","PrePaid"]},"period":{"type":"number"},"period_unit":{"type":"string","enum":["Month"]},"key_pair_name":{"type":"string"},"system_disk_category":{"type":"string"},"network_ref":{"type":"string"},"foundation_stack_name":{"type":"string"},"name_prefix":{"type":"string"},"ops_refs":{"type":"array"},"provider_network_refs":{"type":"object"},"vpc_id":{"type":"string"},"vswitch_id":{"type":"string"},"allocate_eip":{"type":"boolean"},"ingress_cidrs":{"type":"array","items":{"type":"string"}},"default_tags":{"type":"object"}}}`, Description: "复用已有阿里云 基础网络，为 ops bastion 带入 role-based refs，并消费 VPC / vSwitch 这类 provider-specific 网络对象。支持显式选择按量付费或包年包月。", Maturity: "experimental", Capability: "apply_destroy_ready", Enabled: true},
	}

	for _, item := range defaults {
		var existing cloud.DeploymentBlueprint
		err := s.db.Where("code = ?", item.Code).First(&existing).Error
		if err == nil {
			existing.Name = item.Name
			existing.Provider = item.Provider
			existing.Category = item.Category
			existing.Version = item.Version
			existing.TemplatePath = item.TemplatePath
			existing.SchemaJSON = item.SchemaJSON
			existing.Description = item.Description
			existing.Maturity = item.Maturity
			existing.Capability = item.Capability
			existing.Enabled = item.Enabled
			_ = s.db.Save(&existing).Error
			continue
		}
		if err == gorm.ErrRecordNotFound {
			_ = s.db.Create(&item).Error
		}
	}
}
