package project

import (
	"backend-center/internal/domains/shared/scopeutil"
	"encoding/json"
	"errors"
	"strings"

	"gorm.io/gorm"
)

type Service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

func (s *Service) ListProjects() ([]ProjectView, error) {
	var items []Project
	if err := s.db.Order("created_at desc, id desc").Find(&items).Error; err != nil {
		return nil, err
	}
	envCounts := map[uint]int64{}
	stackCounts := map[uint]int64{}

	var envRows []struct {
		ProjectID uint
		Count     int64
	}
	if err := s.db.Model(&Environment{}).Select("project_id, count(*) as count").Group("project_id").Scan(&envRows).Error; err != nil {
		return nil, err
	}
	for _, row := range envRows {
		envCounts[row.ProjectID] = row.Count
	}

	var stackRows []struct {
		ProjectID uint
		Count     int64
	}
	if err := s.db.Model(&Stack{}).Select("project_id, count(*) as count").Group("project_id").Scan(&stackRows).Error; err != nil {
		return nil, err
	}
	for _, row := range stackRows {
		stackCounts[row.ProjectID] = row.Count
	}

	result := make([]ProjectView, 0, len(items))
	for _, item := range items {
		result = append(result, ProjectView{
			ID:               item.ID,
			Code:             item.Code,
			Name:             item.Name,
			Description:      item.Description,
			OwnerUserID:      item.OwnerUserID,
			OwnerTeam:        item.OwnerTeam,
			BusinessLine:     item.BusinessLine,
			Status:           normalizeProjectStatus(item.Status),
			EnvironmentCount: envCounts[item.ID],
			StackCount:       stackCounts[item.ID],
			CreatedAt:        item.CreatedAt,
			UpdatedAt:        item.UpdatedAt,
		})
	}
	return result, nil
}

func (s *Service) CreateProject(input ProjectInput) (ProjectView, error) {
	record := Project{
		Code:         strings.TrimSpace(input.Code),
		Name:         strings.TrimSpace(input.Name),
		Description:  strings.TrimSpace(input.Description),
		OwnerUserID:  input.OwnerUserID,
		OwnerTeam:    strings.TrimSpace(input.OwnerTeam),
		BusinessLine: strings.TrimSpace(input.BusinessLine),
		Status:       normalizeProjectStatus(input.Status),
	}
	if err := validateProject(record); err != nil {
		return ProjectView{}, err
	}
	if err := s.db.Create(&record).Error; err != nil {
		return ProjectView{}, err
	}
	return ProjectView{
		ID:           record.ID,
		Code:         record.Code,
		Name:         record.Name,
		Description:  record.Description,
		OwnerUserID:  record.OwnerUserID,
		OwnerTeam:    record.OwnerTeam,
		BusinessLine: record.BusinessLine,
		Status:       record.Status,
		CreatedAt:    record.CreatedAt,
		UpdatedAt:    record.UpdatedAt,
	}, nil
}

func (s *Service) UpdateProject(id uint, input ProjectInput) (ProjectView, error) {
	var record Project
	if err := s.db.First(&record, id).Error; err != nil {
		return ProjectView{}, err
	}
	record.Code = strings.TrimSpace(input.Code)
	record.Name = strings.TrimSpace(input.Name)
	record.Description = strings.TrimSpace(input.Description)
	record.OwnerUserID = input.OwnerUserID
	record.OwnerTeam = strings.TrimSpace(input.OwnerTeam)
	record.BusinessLine = strings.TrimSpace(input.BusinessLine)
	record.Status = normalizeProjectStatus(input.Status)
	if err := validateProject(record); err != nil {
		return ProjectView{}, err
	}
	if err := s.db.Save(&record).Error; err != nil {
		return ProjectView{}, err
	}
	return ProjectView{
		ID:           record.ID,
		Code:         record.Code,
		Name:         record.Name,
		Description:  record.Description,
		OwnerUserID:  record.OwnerUserID,
		OwnerTeam:    record.OwnerTeam,
		BusinessLine: record.BusinessLine,
		Status:       record.Status,
		CreatedAt:    record.CreatedAt,
		UpdatedAt:    record.UpdatedAt,
	}, nil
}

func (s *Service) DeleteProject(id uint) error {
	var envCount int64
	if err := s.db.Model(&Environment{}).Where("project_id = ?", id).Count(&envCount).Error; err != nil {
		return err
	}
	if envCount > 0 {
		return errors.New("请先删除该项目下的 environments")
	}
	var stackCount int64
	if err := s.db.Model(&Stack{}).Where("project_id = ?", id).Count(&stackCount).Error; err != nil {
		return err
	}
	if stackCount > 0 {
		return errors.New("请先删除该项目下的 stacks")
	}
	return s.db.Delete(&Project{}, id).Error
}

func (s *Service) ListEnvironments(projectID uint) ([]EnvironmentView, error) {
	var projectRecord Project
	if err := s.db.First(&projectRecord, projectID).Error; err != nil {
		return nil, err
	}
	var items []Environment
	if err := s.db.Where("project_id = ?", projectID).Order("created_at asc, id asc").Find(&items).Error; err != nil {
		return nil, err
	}
	stackCounts := map[uint]int64{}
	var rows []struct {
		EnvironmentID uint
		Count         int64
	}
	if err := s.db.Model(&Stack{}).Select("environment_id, count(*) as count").Where("project_id = ?", projectID).Group("environment_id").Scan(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		stackCounts[row.EnvironmentID] = row.Count
	}
	result := make([]EnvironmentView, 0, len(items))
	for _, item := range items {
		result = append(result, EnvironmentView{
			ID:          item.ID,
			ProjectID:   item.ProjectID,
			ProjectCode: projectRecord.Code,
			ProjectName: projectRecord.Name,
			Code:        item.Code,
			Name:        item.Name,
			Kind:        normalizeEnvironmentKind(item.Kind),
			Status:      normalizeProjectStatus(item.Status),
			StackCount:  stackCounts[item.ID],
			CreatedAt:   item.CreatedAt,
			UpdatedAt:   item.UpdatedAt,
		})
	}
	return result, nil
}

func (s *Service) CreateEnvironment(projectID uint, input EnvironmentInput) (EnvironmentView, error) {
	var projectRecord Project
	if err := s.db.First(&projectRecord, projectID).Error; err != nil {
		return EnvironmentView{}, err
	}
	record := Environment{
		ProjectID: projectID,
		Code:      strings.TrimSpace(input.Code),
		Name:      strings.TrimSpace(input.Name),
		Kind:      normalizeEnvironmentKind(input.Kind),
		Status:    normalizeProjectStatus(input.Status),
	}
	if err := validateEnvironment(record); err != nil {
		return EnvironmentView{}, err
	}
	if err := s.db.Create(&record).Error; err != nil {
		return EnvironmentView{}, err
	}
	return EnvironmentView{
		ID:          record.ID,
		ProjectID:   record.ProjectID,
		ProjectCode: projectRecord.Code,
		ProjectName: projectRecord.Name,
		Code:        record.Code,
		Name:        record.Name,
		Kind:        record.Kind,
		Status:      record.Status,
		CreatedAt:   record.CreatedAt,
		UpdatedAt:   record.UpdatedAt,
	}, nil
}

func (s *Service) UpdateEnvironment(projectID, environmentID uint, input EnvironmentInput) (EnvironmentView, error) {
	var record Environment
	if err := s.db.Where("project_id = ?", projectID).First(&record, environmentID).Error; err != nil {
		return EnvironmentView{}, err
	}
	var projectRecord Project
	if err := s.db.First(&projectRecord, projectID).Error; err != nil {
		return EnvironmentView{}, err
	}
	record.Code = strings.TrimSpace(input.Code)
	record.Name = strings.TrimSpace(input.Name)
	record.Kind = normalizeEnvironmentKind(input.Kind)
	record.Status = normalizeProjectStatus(input.Status)
	if err := validateEnvironment(record); err != nil {
		return EnvironmentView{}, err
	}
	if err := s.db.Save(&record).Error; err != nil {
		return EnvironmentView{}, err
	}
	return EnvironmentView{
		ID:          record.ID,
		ProjectID:   record.ProjectID,
		ProjectCode: projectRecord.Code,
		ProjectName: projectRecord.Name,
		Code:        record.Code,
		Name:        record.Name,
		Kind:        record.Kind,
		Status:      record.Status,
		CreatedAt:   record.CreatedAt,
		UpdatedAt:   record.UpdatedAt,
	}, nil
}

func (s *Service) DeleteEnvironment(projectID, environmentID uint) error {
	var stackCount int64
	if err := s.db.Model(&Stack{}).Where("project_id = ? AND environment_id = ?", projectID, environmentID).Count(&stackCount).Error; err != nil {
		return err
	}
	if stackCount > 0 {
		return errors.New("请先删除该环境下的 stacks")
	}
	return s.db.Where("project_id = ?", projectID).Delete(&Environment{}, environmentID).Error
}

func (s *Service) ListStacks(projectID, environmentID *uint) ([]StackView, error) {
	query := s.db.Model(&Stack{})
	if projectID != nil && *projectID > 0 {
		query = query.Where("project_id = ?", *projectID)
	}
	if environmentID != nil && *environmentID > 0 {
		query = query.Where("environment_id = ?", *environmentID)
	}
	var items []Stack
	if err := query.Order("created_at desc, id desc").Find(&items).Error; err != nil {
		return nil, err
	}
	projectNames, projectCodes, environmentNames, environmentCodes, err := s.nameMaps()
	if err != nil {
		return nil, err
	}
	result := make([]StackView, 0, len(items))
	for _, item := range items {
		result = append(result, stackToView(item, projectNames, projectCodes, environmentNames, environmentCodes))
	}
	return result, nil
}

func (s *Service) CreateStack(input StackInput) (StackView, error) {
	record, err := s.buildStackModel(0, input)
	if err != nil {
		return StackView{}, err
	}
	if err := s.db.Create(&record).Error; err != nil {
		return StackView{}, err
	}
	projectNames, projectCodes, environmentNames, environmentCodes, err := s.nameMaps()
	if err != nil {
		return StackView{}, err
	}
	return stackToView(record, projectNames, projectCodes, environmentNames, environmentCodes), nil
}

func (s *Service) UpdateStack(id uint, input StackInput) (StackView, error) {
	record, err := s.buildStackModel(id, input)
	if err != nil {
		return StackView{}, err
	}
	if err := s.db.Save(&record).Error; err != nil {
		return StackView{}, err
	}
	projectNames, projectCodes, environmentNames, environmentCodes, err := s.nameMaps()
	if err != nil {
		return StackView{}, err
	}
	return stackToView(record, projectNames, projectCodes, environmentNames, environmentCodes), nil
}

func (s *Service) DeleteStack(id uint) error {
	return s.db.Delete(&Stack{}, id).Error
}

func (s *Service) buildStackModel(id uint, input StackInput) (Stack, error) {
	var record Stack
	if id > 0 {
		if err := s.db.First(&record, id).Error; err != nil {
			return Stack{}, err
		}
	}
	var env Environment
	if input.ProjectID == 0 || input.EnvironmentID == 0 {
		return Stack{}, errors.New("stack 必须关联 project_id 和 environment_id")
	}
	if err := s.db.Where("project_id = ?", input.ProjectID).First(&env, input.EnvironmentID).Error; err != nil {
		return Stack{}, errors.New("environment 不存在或不属于该 project")
	}
	record.ProjectID = input.ProjectID
	record.EnvironmentID = input.EnvironmentID
	record.Provider = normalizeProjectProvider(input.Provider)
	record.AccountID = input.AccountID
	record.Region = strings.TrimSpace(input.Region)
	record.StackType = normalizeStackType(input.StackType)
	record.StackCode = strings.TrimSpace(input.StackCode)
	record.Name = strings.TrimSpace(input.Name)
	record.Status = normalizeStackStatus(input.Status)
	record.FoundationNetworkID = input.FoundationNetworkID
	record.OwnerMode = normalizeOwnerMode(input.OwnerMode)
	record.BackendType = strings.TrimSpace(input.BackendType)
	record.BackendBucket = strings.TrimSpace(input.BackendBucket)
	record.BackendKey = strings.TrimSpace(input.BackendKey)
	record.BackendLockTable = strings.TrimSpace(input.BackendLockTable)
	record.StateVersion = strings.TrimSpace(input.StateVersion)
	record.CurrentJobID = input.CurrentJobID
	record.LastApplyJobID = input.LastApplyJobID
	record.LastDestroyJobID = input.LastDestroyJobID
	record.DriftStatus = normalizeDriftStatus(input.DriftStatus)
	record.MetadataJSON = encodeMetadata(input.Metadata)
	if err := scopeutil.Validate(s.db, scopeutil.ScopeInput{
		ProjectID:           &record.ProjectID,
		EnvironmentID:       &record.EnvironmentID,
		StackID:             nil,
		FoundationNetworkID: record.FoundationNetworkID,
		AccountID:           record.AccountID,
		Provider:            record.Provider,
	}); err != nil {
		return Stack{}, err
	}
	if err := validateStack(record); err != nil {
		return Stack{}, err
	}
	return record, nil
}

func (s *Service) nameMaps() (map[uint]string, map[uint]string, map[uint]string, map[uint]string, error) {
	projectNames := map[uint]string{}
	projectCodes := map[uint]string{}
	var projects []Project
	if err := s.db.Find(&projects).Error; err != nil {
		return nil, nil, nil, nil, err
	}
	for _, item := range projects {
		projectNames[item.ID] = item.Name
		projectCodes[item.ID] = item.Code
	}
	environmentNames := map[uint]string{}
	environmentCodes := map[uint]string{}
	var environments []Environment
	if err := s.db.Find(&environments).Error; err != nil {
		return nil, nil, nil, nil, err
	}
	for _, item := range environments {
		environmentNames[item.ID] = item.Name
		environmentCodes[item.ID] = item.Code
	}
	return projectNames, projectCodes, environmentNames, environmentCodes, nil
}

func stackToView(item Stack, projectNames, projectCodes, environmentNames, environmentCodes map[uint]string) StackView {
	return StackView{
		ID:                  item.ID,
		ProjectID:           item.ProjectID,
		ProjectCode:         projectCodes[item.ProjectID],
		ProjectName:         projectNames[item.ProjectID],
		EnvironmentID:       item.EnvironmentID,
		EnvironmentCode:     environmentCodes[item.EnvironmentID],
		EnvironmentName:     environmentNames[item.EnvironmentID],
		Provider:            item.Provider,
		AccountID:           item.AccountID,
		Region:              item.Region,
		StackType:           item.StackType,
		StackCode:           item.StackCode,
		Name:                item.Name,
		Status:              item.Status,
		FoundationNetworkID: item.FoundationNetworkID,
		OwnerMode:           item.OwnerMode,
		BackendType:         item.BackendType,
		BackendBucket:       item.BackendBucket,
		BackendKey:          item.BackendKey,
		BackendLockTable:    item.BackendLockTable,
		StateVersion:        item.StateVersion,
		CurrentJobID:        item.CurrentJobID,
		LastApplyJobID:      item.LastApplyJobID,
		LastDestroyJobID:    item.LastDestroyJobID,
		DriftStatus:         item.DriftStatus,
		Metadata:            decodeMetadata(item.MetadataJSON),
		CreatedAt:           item.CreatedAt,
		UpdatedAt:           item.UpdatedAt,
	}
}

func validateProject(item Project) error {
	if item.Code == "" {
		return errors.New("project code 不能为空")
	}
	if item.Name == "" {
		return errors.New("project name 不能为空")
	}
	return nil
}

func validateEnvironment(item Environment) error {
	if item.Code == "" {
		return errors.New("environment code 不能为空")
	}
	if item.Name == "" {
		return errors.New("environment name 不能为空")
	}
	return nil
}

func validateStack(item Stack) error {
	if item.Provider == "" {
		return errors.New("stack provider 不能为空")
	}
	if item.Region == "" {
		return errors.New("stack region 不能为空")
	}
	if item.StackType == "" {
		return errors.New("stack_type 不支持")
	}
	if item.StackCode == "" {
		return errors.New("stack_code 不能为空")
	}
	if item.Name == "" {
		return errors.New("stack name 不能为空")
	}
	return nil
}

func normalizeProjectStatus(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "active", "disabled", "archived":
		return strings.TrimSpace(strings.ToLower(value))
	default:
		return "active"
	}
}

func normalizeEnvironmentKind(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "dev", "test", "staging", "prod":
		return strings.TrimSpace(strings.ToLower(value))
	default:
		return "dev"
	}
}

func normalizeStackType(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "foundation-network", "access-node", "compute-batch", "cluster", "shared-service":
		return strings.TrimSpace(strings.ToLower(value))
	default:
		return ""
	}
}

func normalizeProjectProvider(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "aws":
		return "aws"
	case "ali", "alicloud":
		return "alicloud"
	default:
		return ""
	}
}

func normalizeStackStatus(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "draft", "planned", "applied", "active", "drifted", "retiring", "destroyed", "orphaned":
		return strings.TrimSpace(strings.ToLower(value))
	default:
		return "draft"
	}
}

func normalizeOwnerMode(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "terraform", "api", "hybrid":
		return strings.TrimSpace(strings.ToLower(value))
	default:
		return "terraform"
	}
}

func normalizeDriftStatus(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "clean", "drifted", "unknown":
		return strings.TrimSpace(strings.ToLower(value))
	default:
		return "clean"
	}
}

func encodeMetadata(value map[string]any) string {
	if len(value) == 0 {
		return ""
	}
	raw, _ := json.Marshal(value)
	return string(raw)
}

func decodeMetadata(raw string) map[string]any {
	if strings.TrimSpace(raw) == "" {
		return map[string]any{}
	}
	var value map[string]any
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		return map[string]any{}
	}
	return value
}
