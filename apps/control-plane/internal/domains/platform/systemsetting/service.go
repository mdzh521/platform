package systemsetting

import "gorm.io/gorm"

type Service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

func (s *Service) List() ([]SystemSetting, error) {
	var items []SystemSetting
	err := s.db.Order("id asc").Find(&items).Error
	return items, err
}

func (s *Service) Upsert(input SystemSetting) (*SystemSetting, error) {
	var model SystemSetting
	err := s.db.Where("`key` = ?", input.Key).First(&model).Error
	if err == gorm.ErrRecordNotFound {
		if err := s.db.Create(&input).Error; err != nil {
			return nil, err
		}
		return &input, nil
	}
	if err != nil {
		return nil, err
	}

	model.Value = input.Value
	model.Description = input.Description
	if err := s.db.Save(&model).Error; err != nil {
		return nil, err
	}
	return &model, nil
}

func (s *Service) Delete(id uint) error {
	return s.db.Delete(&SystemSetting{}, id).Error
}
