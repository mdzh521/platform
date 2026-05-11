package cloud

import "gorm.io/gorm"

type namedModel interface {
	GetID() uint
	GetName() string
}

func (m CloudAccount) GetID() uint            { return m.ID }
func (m CloudAccount) GetName() string        { return m.Name }
func (m NetworkPlan) GetID() uint             { return m.ID }
func (m NetworkPlan) GetName() string         { return m.Name }
func (m DeploymentBlueprint) GetID() uint     { return m.ID }
func (m DeploymentBlueprint) GetName() string { return m.Name }

func loadNameMap[T namedModel](db *gorm.DB) (map[uint]string, error) {
	var items []T
	if err := db.Find(&items).Error; err != nil {
		return nil, err
	}
	result := make(map[uint]string, len(items))
	for _, item := range items {
		result[item.GetID()] = item.GetName()
	}
	return result, nil
}

func loadJobResourceCounts(db *gorm.DB) (map[uint]int64, error) {
	type row struct {
		JobID uint
		Count int64
	}
	var rows []row
	if err := db.Model(&DeploymentResource{}).Select("job_id, count(*) as count").Group("job_id").Scan(&rows).Error; err != nil {
		return nil, err
	}
	result := make(map[uint]int64, len(rows))
	for _, item := range rows {
		result[item.JobID] = item.Count
	}
	return result, nil
}

func loadJobResources(db *gorm.DB, jobID uint) ([]DeploymentResourceView, error) {
	var items []DeploymentResource
	if err := db.Where("job_id = ?", jobID).Order("resource_type asc, resource_name asc, id asc").Find(&items).Error; err != nil {
		return nil, err
	}
	result := make([]DeploymentResourceView, 0, len(items))
	for _, item := range items {
		result = append(result, resourceToView(item))
	}
	return result, nil
}
