package summary

import (
	"backend-center/internal/domains/delivery/cloud"

	"gorm.io/gorm"
)

type SummaryService struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) *SummaryService {
	return &SummaryService{db: db}
}

func (s *SummaryService) Summary() (cloud.Summary, error) {
	var result cloud.Summary
	if err := s.db.Model(&cloud.CloudAccount{}).Count(&result.Accounts).Error; err != nil {
		return result, err
	}
	if err := s.db.Model(&cloud.NetworkPlan{}).Count(&result.Networks).Error; err != nil {
		return result, err
	}
	if err := s.db.Model(&cloud.DeploymentBlueprint{}).Where("enabled = ?", true).Count(&result.Blueprints).Error; err != nil {
		return result, err
	}
	if err := s.db.Model(&cloud.DeploymentJob{}).Count(&result.Jobs).Error; err != nil {
		return result, err
	}
	if err := s.db.Model(&cloud.DeploymentJob{}).Where("status IN ?", []string{"queued", "planning", "applying"}).Count(&result.QueuedJobs).Error; err != nil {
		return result, err
	}
	if err := s.db.Model(&cloud.CloudResource{}).Count(&result.Resources).Error; err != nil {
		return result, err
	}
	return result, nil
}
