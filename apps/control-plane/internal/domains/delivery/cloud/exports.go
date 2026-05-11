package cloud

import (
	platformqueue "backend-center/internal/infra/queue"
	"backend-center/internal/infra/secure"

	"gorm.io/gorm"
)

type Dependencies struct {
	DB     *gorm.DB
	Queue  platformqueue.Dispatcher
	Cipher *secure.Cipher
}

type baseService struct {
	db     *gorm.DB
	queue  platformqueue.Dispatcher
	cipher *secure.Cipher
}

func NewDependencies(db *gorm.DB, queue platformqueue.Dispatcher, secret string) *Dependencies {
	return &Dependencies{
		DB:     db,
		Queue:  queue,
		Cipher: secure.New(secret),
	}
}

func newBaseService(dep *Dependencies) *baseService {
	return &baseService{
		db:     dep.DB,
		queue:  dep.Queue,
		cipher: dep.Cipher,
	}
}

func NewAccountService(dep *Dependencies) *AccountService {
	return &AccountService{base: newBaseService(dep)}
}

func NewNetworkPlanService(dep *Dependencies) *NetworkPlanService {
	return &NetworkPlanService{base: newBaseService(dep)}
}

func BlueprintSupportsApply(capability string) bool {
	return blueprintSupportsApply(capability)
}

func BlueprintSupportsDestroy(capability string) bool {
	return blueprintSupportsDestroy(capability)
}
