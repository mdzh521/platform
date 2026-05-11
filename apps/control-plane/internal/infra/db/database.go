package database

import (
	"backend-center/internal/config"
	"backend-center/internal/domains/delivery/cloud"
	"backend-center/internal/domains/graph"
	"backend-center/internal/domains/iam/identitysource"
	"backend-center/internal/domains/clusters/k8s"
	"backend-center/internal/domains/machines/machine"
	"backend-center/internal/domains/projects/project"
	"backend-center/internal/domains/iam/rbac"
	"backend-center/internal/domains/platform/systemsetting"
	"backend-center/internal/domains/iam/user"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func Open(cfg config.Config) (*gorm.DB, error) {
	return gorm.Open(mysql.Open(cfg.DSN()), &gorm.Config{})
}

func Migrate(db *gorm.DB) error {
	if err := db.AutoMigrate(
		&user.User{},
		&user.Credential{},
		&identitysource.IdentitySource{},
		&user.ExternalIdentity{},
		&user.Group{},
		&user.GroupMember{},
		&rbac.Role{},
		&rbac.Permission{},
		&rbac.RolePermission{},
		&rbac.SubjectRoleBinding{},
		&systemsetting.SystemSetting{},
		&project.Project{},
		&project.Environment{},
		&project.Stack{},
		&cloud.CloudAccount{},
		&cloud.NetworkPlan{},
		&cloud.DeploymentBlueprint{},
		&cloud.DeploymentJob{},
		&cloud.DeploymentJobLog{},
		&cloud.DeploymentResource{},
		&cloud.CloudResource{},
		&cloud.ClusterAddonExecution{},
		&graph.Node{},
		&graph.Edge{},
		&k8s.Cluster{},
		&k8s.Namespace{},
		&k8s.Workload{},
		&machine.AssetGroup{},
		&machine.Asset{},
		&machine.Account{},
		&machine.CredentialLibrary{},
		&machine.QuickCommand{},
		&machine.Session{},
		&machine.SessionEvent{},
		&machine.SSHHostTrust{},
	); err != nil {
		return err
	}
	return hardenSchema(db)
}
