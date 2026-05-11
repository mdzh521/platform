package database

import (
	"fmt"

	"gorm.io/gorm"
)

type foreignKeySpec struct {
	table     string
	name      string
	column    string
	refTable  string
	refColumn string
	onDelete  string
	onUpdate  string
}

type indexSpec struct {
	table   string
	name    string
	columns string
	unique  bool
}

func hardenSchema(db *gorm.DB) error {
	for _, spec := range foreignKeySpecs() {
		if err := addForeignKeyIfMissing(db, spec); err != nil {
			return err
		}
	}
	for _, spec := range indexSpecs() {
		if err := addIndexIfMissing(db, spec); err != nil {
			return err
		}
	}
	return nil
}

func foreignKeySpecs() []foreignKeySpec {
	return []foreignKeySpec{
		{table: "environments", name: "fk_environments_project", column: "project_id", refTable: "projects", refColumn: "id", onDelete: "RESTRICT", onUpdate: "CASCADE"},
		{table: "stacks", name: "fk_stacks_project", column: "project_id", refTable: "projects", refColumn: "id", onDelete: "RESTRICT", onUpdate: "CASCADE"},
		{table: "stacks", name: "fk_stacks_environment", column: "environment_id", refTable: "environments", refColumn: "id", onDelete: "RESTRICT", onUpdate: "CASCADE"},
		{table: "cloud_accounts", name: "fk_cloud_accounts_default_project", column: "default_project_id", refTable: "projects", refColumn: "id", onDelete: "SET NULL", onUpdate: "CASCADE"},
		{table: "cloud_accounts", name: "fk_cloud_accounts_default_environment", column: "default_environment_id", refTable: "environments", refColumn: "id", onDelete: "SET NULL", onUpdate: "CASCADE"},
		{table: "network_plans", name: "fk_network_plans_account", column: "account_id", refTable: "cloud_accounts", refColumn: "id", onDelete: "RESTRICT", onUpdate: "CASCADE"},
		{table: "network_plans", name: "fk_network_plans_project", column: "project_id", refTable: "projects", refColumn: "id", onDelete: "RESTRICT", onUpdate: "CASCADE"},
		{table: "network_plans", name: "fk_network_plans_environment", column: "environment_id", refTable: "environments", refColumn: "id", onDelete: "RESTRICT", onUpdate: "CASCADE"},
		{table: "network_plans", name: "fk_network_plans_stack", column: "stack_id", refTable: "stacks", refColumn: "id", onDelete: "RESTRICT", onUpdate: "CASCADE"},
		{table: "deployment_jobs", name: "fk_deployment_jobs_account", column: "account_id", refTable: "cloud_accounts", refColumn: "id", onDelete: "RESTRICT", onUpdate: "CASCADE"},
		{table: "deployment_jobs", name: "fk_deployment_jobs_blueprint", column: "blueprint_id", refTable: "deployment_blueprints", refColumn: "id", onDelete: "RESTRICT", onUpdate: "CASCADE"},
		{table: "deployment_jobs", name: "fk_deployment_jobs_network_plan", column: "network_plan_id", refTable: "network_plans", refColumn: "id", onDelete: "RESTRICT", onUpdate: "CASCADE"},
		{table: "deployment_jobs", name: "fk_deployment_jobs_project", column: "project_id", refTable: "projects", refColumn: "id", onDelete: "RESTRICT", onUpdate: "CASCADE"},
		{table: "deployment_jobs", name: "fk_deployment_jobs_environment", column: "environment_id", refTable: "environments", refColumn: "id", onDelete: "RESTRICT", onUpdate: "CASCADE"},
		{table: "deployment_jobs", name: "fk_deployment_jobs_stack", column: "stack_id", refTable: "stacks", refColumn: "id", onDelete: "RESTRICT", onUpdate: "CASCADE"},
		{table: "deployment_job_logs", name: "fk_deployment_job_logs_job", column: "job_id", refTable: "deployment_jobs", refColumn: "id", onDelete: "CASCADE", onUpdate: "CASCADE"},
		{table: "deployment_resources", name: "fk_deployment_resources_job", column: "job_id", refTable: "deployment_jobs", refColumn: "id", onDelete: "CASCADE", onUpdate: "CASCADE"},
		{table: "cloud_resources", name: "fk_cloud_resources_account", column: "account_id", refTable: "cloud_accounts", refColumn: "id", onDelete: "RESTRICT", onUpdate: "CASCADE"},
		{table: "cloud_resources", name: "fk_cloud_resources_blueprint", column: "blueprint_id", refTable: "deployment_blueprints", refColumn: "id", onDelete: "RESTRICT", onUpdate: "CASCADE"},
		{table: "cloud_resources", name: "fk_cloud_resources_network_plan", column: "network_plan_id", refTable: "network_plans", refColumn: "id", onDelete: "RESTRICT", onUpdate: "CASCADE"},
		{table: "cloud_resources", name: "fk_cloud_resources_project", column: "project_id", refTable: "projects", refColumn: "id", onDelete: "RESTRICT", onUpdate: "CASCADE"},
		{table: "cloud_resources", name: "fk_cloud_resources_environment", column: "environment_id", refTable: "environments", refColumn: "id", onDelete: "RESTRICT", onUpdate: "CASCADE"},
		{table: "cloud_resources", name: "fk_cloud_resources_stack", column: "stack_id", refTable: "stacks", refColumn: "id", onDelete: "RESTRICT", onUpdate: "CASCADE"},
		{table: "cloud_resources", name: "fk_cloud_resources_source_job", column: "source_job_id", refTable: "deployment_jobs", refColumn: "id", onDelete: "RESTRICT", onUpdate: "CASCADE"},
		{table: "cluster_addon_executions", name: "fk_cluster_addon_executions_cluster", column: "cluster_id", refTable: "clusters", refColumn: "id", onDelete: "CASCADE", onUpdate: "CASCADE"},
		{table: "cluster_addon_executions", name: "fk_cluster_addon_executions_job", column: "source_job_id", refTable: "deployment_jobs", refColumn: "id", onDelete: "RESTRICT", onUpdate: "CASCADE"},
		{table: "cluster_addon_executions", name: "fk_cluster_addon_executions_resource", column: "resource_id", refTable: "cloud_resources", refColumn: "id", onDelete: "RESTRICT", onUpdate: "CASCADE"},
		{table: "cluster_addon_executions", name: "fk_cluster_addon_executions_account", column: "account_id", refTable: "cloud_accounts", refColumn: "id", onDelete: "RESTRICT", onUpdate: "CASCADE"},
		{table: "clusters", name: "fk_clusters_project", column: "project_id", refTable: "projects", refColumn: "id", onDelete: "RESTRICT", onUpdate: "CASCADE"},
		{table: "clusters", name: "fk_clusters_environment", column: "environment_id", refTable: "environments", refColumn: "id", onDelete: "RESTRICT", onUpdate: "CASCADE"},
		{table: "clusters", name: "fk_clusters_stack", column: "stack_id", refTable: "stacks", refColumn: "id", onDelete: "RESTRICT", onUpdate: "CASCADE"},
		{table: "clusters", name: "fk_clusters_foundation_network", column: "foundation_network_id", refTable: "network_plans", refColumn: "id", onDelete: "RESTRICT", onUpdate: "CASCADE"},
		{table: "namespaces", name: "fk_namespaces_cluster", column: "cluster_id", refTable: "clusters", refColumn: "id", onDelete: "CASCADE", onUpdate: "CASCADE"},
		{table: "workloads", name: "fk_workloads_cluster", column: "cluster_id", refTable: "clusters", refColumn: "id", onDelete: "CASCADE", onUpdate: "CASCADE"},
		{table: "workloads", name: "fk_workloads_namespace", column: "namespace_id", refTable: "namespaces", refColumn: "id", onDelete: "SET NULL", onUpdate: "CASCADE"},
		{table: "assets", name: "fk_assets_project", column: "project_id", refTable: "projects", refColumn: "id", onDelete: "RESTRICT", onUpdate: "CASCADE"},
		{table: "assets", name: "fk_assets_environment", column: "environment_id", refTable: "environments", refColumn: "id", onDelete: "RESTRICT", onUpdate: "CASCADE"},
		{table: "assets", name: "fk_assets_stack", column: "stack_id", refTable: "stacks", refColumn: "id", onDelete: "RESTRICT", onUpdate: "CASCADE"},
		{table: "assets", name: "fk_assets_foundation_network", column: "foundation_network_id", refTable: "network_plans", refColumn: "id", onDelete: "RESTRICT", onUpdate: "CASCADE"},
		{table: "assets", name: "fk_assets_gateway_asset", column: "gateway_asset_id", refTable: "assets", refColumn: "id", onDelete: "SET NULL", onUpdate: "CASCADE"},
		{table: "assets", name: "fk_assets_source_job", column: "source_job_id", refTable: "deployment_jobs", refColumn: "id", onDelete: "RESTRICT", onUpdate: "CASCADE"},
		{table: "accounts", name: "fk_accounts_asset", column: "asset_id", refTable: "assets", refColumn: "id", onDelete: "CASCADE", onUpdate: "CASCADE"},
		{table: "credential_libraries", name: "fk_credential_libraries_asset", column: "asset_id", refTable: "assets", refColumn: "id", onDelete: "SET NULL", onUpdate: "CASCADE"},
	}
}

func indexSpecs() []indexSpec {
	return []indexSpec{
		{table: "cloud_resources", name: "idx_cloud_resources_job_type_cloud", columns: "source_job_id, resource_type, cloud_id"},
		{table: "assets", name: "idx_assets_source_provider_resource", columns: "source_provider, source_resource_id"},
		{table: "clusters", name: "idx_clusters_provider_source_resource", columns: "provider, source_resource_id"},
		{table: "deployment_jobs", name: "idx_deployment_jobs_scope_status", columns: "network_plan_id, project_id, environment_id, stack_id, status"},
	}
}

func addForeignKeyIfMissing(db *gorm.DB, spec foreignKeySpec) error {
	var count int64
	query := `
SELECT COUNT(*) 
FROM information_schema.TABLE_CONSTRAINTS
WHERE CONSTRAINT_SCHEMA = DATABASE()
  AND TABLE_NAME = ?
  AND CONSTRAINT_NAME = ?
  AND CONSTRAINT_TYPE = 'FOREIGN KEY'`
	if err := db.Raw(query, spec.table, spec.name).Scan(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	statement := fmt.Sprintf(
		"ALTER TABLE `%s` ADD CONSTRAINT `%s` FOREIGN KEY (`%s`) REFERENCES `%s`(`%s`) ON DELETE %s ON UPDATE %s",
		spec.table, spec.name, spec.column, spec.refTable, spec.refColumn, spec.onDelete, spec.onUpdate,
	)
	return db.Exec(statement).Error
}

func addIndexIfMissing(db *gorm.DB, spec indexSpec) error {
	var count int64
	query := `
SELECT COUNT(*)
FROM information_schema.STATISTICS
WHERE TABLE_SCHEMA = DATABASE()
  AND TABLE_NAME = ?
  AND INDEX_NAME = ?`
	if err := db.Raw(query, spec.table, spec.name).Scan(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	indexType := "INDEX"
	if spec.unique {
		indexType = "UNIQUE INDEX"
	}
	statement := fmt.Sprintf("ALTER TABLE `%s` ADD %s `%s` (%s)", spec.table, indexType, spec.name, spec.columns)
	return db.Exec(statement).Error
}
