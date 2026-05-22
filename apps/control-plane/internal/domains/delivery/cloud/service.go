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
	Workbench   *WorkbenchService
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
	accounts := &AccountService{base: core}
	networks := &NetworkPlanService{base: core}
	deployments := &DeploymentService{base: core}
	resources := &ResourceService{base: core}
	service := &Service{
		base:        core,
		Summary:     summary,
		Accounts:    accounts,
		Networks:    networks,
		Blueprints:  blueprints,
		Deployments: deployments,
		Resources:   resources,
		Addons:      &AddonService{base: core},
		Workbench: &WorkbenchService{
			summary:     summary,
			accounts:    accounts,
			networks:    networks,
			blueprints:  blueprints,
			deployments: deployments,
			resources:   resources,
		},
	}
	if service.Blueprints != nil {
		service.Blueprints.EnsureDefaultBlueprints()
	}
	return service
}
