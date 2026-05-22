package cloud

import (
	"backend-center/internal/domains/shared/scopeutil"
	"errors"
	"fmt"
	"net"
	"strings"
)

type NetworkPlanService struct {
	base *baseService
}

func (s *NetworkPlanService) List() ([]NetworkPlanView, error) {
	var items []NetworkPlan
	if err := s.base.db.Order("created_at desc").Find(&items).Error; err != nil {
		return nil, err
	}
	accountNames, err := loadNameMap[CloudAccount](s.base.db)
	if err != nil {
		return nil, err
	}
	result := make([]NetworkPlanView, 0, len(items))
	for _, item := range items {
		result = append(result, networkToView(item, accountNames[item.AccountID]))
	}
	return result, nil
}

func (s *NetworkPlanService) Create(input NetworkPlanInput) (NetworkPlanView, error) {
	return s.save(0, input)
}

func (s *NetworkPlanService) Update(id uint, input NetworkPlanInput) (NetworkPlanView, error) {
	if id == 0 {
		return NetworkPlanView{}, errors.New("无效的网络规划 ID")
	}
	return s.save(id, input)
}

func (s *NetworkPlanService) Delete(id uint) error {
	if id == 0 {
		return errors.New("无效的网络规划 ID")
	}

	var item NetworkPlan
	if err := s.base.db.First(&item, id).Error; err != nil {
		return err
	}

	var jobCount int64
	if err := s.base.db.Model(&DeploymentJob{}).Where("network_plan_id = ?", id).Count(&jobCount).Error; err != nil {
		return err
	}
	if jobCount > 0 {
		return fmt.Errorf("请先清理引用该基础网络的基础交付任务（共 %d 条）", jobCount)
	}

	var resourceCount int64
	if err := s.base.db.Model(&CloudResource{}).Where("network_plan_id = ?", id).Count(&resourceCount).Error; err != nil {
		return err
	}
	if resourceCount > 0 {
		return fmt.Errorf("请先清理引用该基础网络的资源台账（共 %d 条）", resourceCount)
	}

	return s.base.db.Delete(&item).Error
}

func (s *NetworkPlanService) save(id uint, input NetworkPlanInput) (NetworkPlanView, error) {
	if strings.TrimSpace(input.Name) == "" {
		return NetworkPlanView{}, errors.New("网络规划名称不能为空")
	}
	provider := normalizeProvider(input.Provider)
	if provider == "" {
		return NetworkPlanView{}, errors.New("请选择云平台")
	}
	if input.AccountID == 0 {
		return NetworkPlanView{}, errors.New("请选择云账号")
	}
	if strings.TrimSpace(input.Region) == "" {
		return NetworkPlanView{}, errors.New("区域不能为空")
	}
	if err := scopeutil.Validate(s.base.db, scopeutil.ScopeInput{
		AccountID:     &input.AccountID,
		ProjectID:     input.ProjectID,
		EnvironmentID: input.EnvironmentID,
		StackID:       input.StackID,
		Provider:      provider,
	}); err != nil {
		return NetworkPlanView{}, err
	}
	if _, _, err := net.ParseCIDR(strings.TrimSpace(input.VPCCIDR)); err != nil {
		return NetworkPlanView{}, errors.New("VPC CIDR 格式不正确")
	}
	subnetGroups := normalizeSubnetGroups(input, normalizeField(input.VPCName, deriveVPCName(input.Name, normalizeField(input.Environment, "dev"))))
	publicSubnets := flattenSubnetCIDRs(subnetGroups, "public")
	privateSubnets := flattenSubnetCIDRs(subnetGroups, "private")
	if len(subnetGroups) == 0 {
		return NetworkPlanView{}, errors.New("请至少配置一个子网分组")
	}
	switch provider {
	case "aws":
		if len(publicSubnets) == 0 || len(privateSubnets) == 0 {
			return NetworkPlanView{}, errors.New("AWS 网络规划至少要同时包含公网和私网子网分组")
		}
	case "alicloud":
		if !hasRole(subnetGroups, "slb") {
			return NetworkPlanView{}, errors.New("阿里云网络规划至少要包含一个入口角色，如 slb")
		}
		if !hasTrafficProfiles(subnetGroups, "workload", "cluster-node", "data", "ops", "pod-network") {
			return NetworkPlanView{}, errors.New("阿里云网络规划至少要包含一个 workload / cluster-node / data / ops / pod-network 类角色")
		}
	}
	allCIDRs := []string{}
	for _, group := range subnetGroups {
		allCIDRs = append(allCIDRs, extractStringSlice(group["cidrs"])...)
	}
	if bastionCIDR := strings.TrimSpace(input.BastionSubnetCIDR); bastionCIDR != "" {
		allCIDRs = append(allCIDRs, bastionCIDR)
	}
	for _, cidr := range allCIDRs {
		if _, _, err := net.ParseCIDR(strings.TrimSpace(cidr)); err != nil {
			return NetworkPlanView{}, errors.New("子网 CIDR 格式不正确")
		}
	}

	model := NetworkPlan{
		Name:          strings.TrimSpace(input.Name),
		Provider:      provider,
		AccountID:     input.AccountID,
		ProjectID:     input.ProjectID,
		EnvironmentID: input.EnvironmentID,
		StackID:       input.StackID,
		Region:        strings.TrimSpace(input.Region),
		VPCCIDR:       strings.TrimSpace(input.VPCCIDR),
		TopologyJSON:  mustJSONString(buildTopology(input)),
		Status:        "planned",
	}
	if id == 0 {
		if err := s.base.db.Create(&model).Error; err != nil {
			return NetworkPlanView{}, err
		}
		if err := bindNetworkPlanScope(s.base.db, &model); err != nil {
			return NetworkPlanView{}, err
		}
	} else {
		var existing NetworkPlan
		if err := s.base.db.First(&existing, id).Error; err != nil {
			return NetworkPlanView{}, err
		}
		existing.Name = model.Name
		existing.Provider = model.Provider
		existing.AccountID = model.AccountID
		existing.ProjectID = model.ProjectID
		existing.EnvironmentID = model.EnvironmentID
		existing.StackID = model.StackID
		existing.Region = model.Region
		existing.VPCCIDR = model.VPCCIDR
		existing.TopologyJSON = model.TopologyJSON
		existing.Status = "planned"
		if err := s.base.db.Save(&existing).Error; err != nil {
			return NetworkPlanView{}, err
		}
		model = existing
		if err := bindNetworkPlanScope(s.base.db, &model); err != nil {
			return NetworkPlanView{}, err
		}
	}

	var account CloudAccount
	_ = s.base.db.First(&account, input.AccountID).Error
	return networkToView(model, account.Name), nil
}

func hasRole(groups []map[string]any, wanted string) bool {
	wanted = strings.TrimSpace(strings.ToLower(wanted))
	for _, group := range groups {
		if strings.TrimSpace(strings.ToLower(fmt.Sprint(group["role"]))) == wanted {
			return true
		}
	}
	return false
}

func hasTrafficProfiles(groups []map[string]any, profiles ...string) bool {
	wanted := map[string]struct{}{}
	for _, profile := range profiles {
		profile = strings.TrimSpace(strings.ToLower(profile))
		if profile != "" {
			wanted[profile] = struct{}{}
		}
	}
	for _, group := range groups {
		profile := strings.TrimSpace(strings.ToLower(fmt.Sprint(group["traffic_profile"])))
		if _, ok := wanted[profile]; ok {
			return true
		}
	}
	return false
}
