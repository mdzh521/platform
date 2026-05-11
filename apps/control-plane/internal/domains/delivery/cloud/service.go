package cloud

type Service struct {
	base        *baseService
	Summary     SummaryOperations
	Accounts    *AccountService
	Networks    *NetworkPlanService
	Blueprints  BlueprintOperations
	Deployments *DeploymentService
	Resources   *ResourceService
	Addons      *AddonService
}

type SummaryOperations interface {
	Summary() (Summary, error)
}

type BlueprintOperations interface {
	List() ([]BlueprintView, error)
	EnsureDefaultBlueprints()
}

func NewService(base *Dependencies, summary SummaryOperations, blueprints BlueprintOperations) *Service {
	core := newBaseService(base)
	service := &Service{
		base:        core,
		Summary:     summary,
		Accounts:    &AccountService{base: core},
		Networks:    &NetworkPlanService{base: core},
		Blueprints:  blueprints,
		Deployments: &DeploymentService{base: core},
		Resources:   &ResourceService{base: core},
		Addons:      &AddonService{base: core},
	}
	if service.Blueprints != nil {
		service.Blueprints.EnsureDefaultBlueprints()
	}
	return service
}
