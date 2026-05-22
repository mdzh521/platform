package http

import (
	"context"
	nethttp "net/http"
	"os"
	"path/filepath"
	"strings"

	"backend-center/internal/config"
	"backend-center/internal/domains/clusters/k8s"
	"backend-center/internal/domains/delivery/cloud"
	"backend-center/internal/domains/graph"
	"backend-center/internal/domains/iam/auth"
	"backend-center/internal/domains/iam/identitysource"
	"backend-center/internal/domains/iam/rbac"
	"backend-center/internal/domains/iam/user"
	"backend-center/internal/domains/machines/machine"
	"backend-center/internal/domains/platform/systemsetting"
	"backend-center/internal/domains/projects/project"
	platformruntime "backend-center/internal/infra/runtime"
	"backend-center/internal/infra/web/middleware"

	"github.com/gin-gonic/gin"
)

type Dependencies struct {
	Config                config.Config
	Infra                 *platformruntime.Infra
	AuthHandler           *auth.Handler
	UserHandler           *user.Handler
	IdentitySourceHandler *identitysource.Handler
	RBACHandler           *rbac.Handler
	SystemSettingHandler  *systemsetting.Handler
	ProjectHandler        *project.Handler
	GraphHandler          *graph.Handler
	CloudHandler          *cloud.Handler
	K8sHandler            *k8s.Handler
	MachineHandler        *machine.Handler
}

func NewRouter(dep Dependencies) *gin.Engine {
	router := gin.Default()
	healthReport := func() platformruntime.HealthReport {
		if dep.Infra == nil {
			return platformruntime.HealthReport{
				Status: "degraded",
				Cache:  platformruntime.DriverHealth{Driver: "disabled", Healthy: false, Error: "infra not configured"},
				Queue:  platformruntime.DriverHealth{Driver: "disabled", Healthy: false, Error: "infra not configured"},
			}
		}
		return dep.Infra.Health(context.Background())
	}

	if dep.Config.Frontend.Mode == "embedded" {
		router.Use(func(c *gin.Context) {
			if c.Request.URL.Path == "/" {
				c.Header("Cache-Control", "no-store, no-cache, must-revalidate")
				c.Header("Pragma", "no-cache")
				c.Header("Expires", "0")
			}
			c.Next()
		})

		router.GET("/", func(c *gin.Context) {
			c.File(filepath.Clean("./web/index.html"))
		})
		router.NoRoute(func(c *gin.Context) {
			if strings.HasPrefix(c.Request.URL.Path, "/api/") || c.Request.URL.Path == "/healthz" {
				c.JSON(nethttp.StatusNotFound, gin.H{"error": "not found"})
				return
			}
			if _, err := os.Stat("./web/index.html"); err == nil {
				c.File(filepath.Clean("./web/index.html"))
				return
			}
			c.JSON(nethttp.StatusNotFound, gin.H{"error": "not found"})
		})
	} else {
		router.GET("/", func(c *gin.Context) {
			c.JSON(nethttp.StatusOK, gin.H{
				"data": gin.H{
					"service": "backend-center-api",
					"status":  "ok",
				},
			})
		})
		router.NoRoute(func(c *gin.Context) {
			c.JSON(nethttp.StatusNotFound, gin.H{"error": "not found"})
		})
	}

	router.GET("/healthz", func(c *gin.Context) {
		report := healthReport()
		statusCode := nethttp.StatusOK
		if report.Status != "ok" {
			statusCode = nethttp.StatusServiceUnavailable
		}
		c.JSON(statusCode, report)
	})

	api := router.Group("/api/v1")
	{
		api.POST("/auth/login/local", dep.AuthHandler.LoginLocal)
		api.GET("/k8s/workloads/:id/terminal", dep.K8sHandler.OpenInteractiveTerminal)
		api.GET("/runtime/config", func(c *gin.Context) {
			runtimeCfg := dep.Config.Runtime()
			c.JSON(nethttp.StatusOK, gin.H{
				"data": gin.H{
					"app_name": runtimeCfg.AppName,
					"run_mode": runtimeCfg.RunMode,
					"frontend": runtimeCfg.Frontend,
					"cache":    runtimeCfg.Cache,
					"queue":    runtimeCfg.Queue,
					"health":   healthReport(),
				},
			})
		})
	}

	protected := api.Group("/")
	protected.Use(middleware.JWT(dep.Config.JWTSecret))
	{
		protected.GET("/users", dep.UserHandler.List)
		protected.POST("/users", dep.UserHandler.CreateLocal)
		protected.GET("/users/:id", dep.UserHandler.Get)
		protected.PUT("/users/:id", dep.UserHandler.Update)
		protected.PUT("/users/:id/password", dep.UserHandler.ResetPassword)
		protected.DELETE("/users/:id", dep.UserHandler.Delete)
		protected.POST("/users/:id/disable", dep.UserHandler.Disable)
		protected.GET("/groups", dep.UserHandler.ListGroups)
		protected.POST("/groups", dep.UserHandler.CreateGroup)
		protected.PUT("/groups/:id", dep.UserHandler.UpdateGroup)
		protected.DELETE("/groups/:id", dep.UserHandler.DeleteGroup)
		protected.POST("/groups/:id/members", dep.UserHandler.AddGroupMember)
		protected.DELETE("/groups/:id/members/:userId", dep.UserHandler.RemoveGroupMember)

		protected.GET("/identity-sources", dep.IdentitySourceHandler.List)
		protected.POST("/identity-sources", dep.IdentitySourceHandler.Create)
		protected.PUT("/identity-sources/:id", dep.IdentitySourceHandler.Update)
		protected.DELETE("/identity-sources/:id", dep.IdentitySourceHandler.Delete)
		protected.POST("/identity-sources/:id/test", dep.IdentitySourceHandler.Test)

		protected.GET("/roles", dep.RBACHandler.ListRoles)
		protected.POST("/roles", dep.RBACHandler.CreateRole)
		protected.PUT("/roles/:id", dep.RBACHandler.UpdateRole)
		protected.DELETE("/roles/:id", dep.RBACHandler.DeleteRole)
		protected.GET("/role-bindings", dep.RBACHandler.ListBindings)
		protected.POST("/role-bindings", dep.RBACHandler.AssignRole)
		protected.DELETE("/role-bindings/:id", dep.RBACHandler.DeleteBinding)

		protected.GET("/system-settings", dep.SystemSettingHandler.List)
		protected.POST("/system-settings", dep.SystemSettingHandler.Upsert)
		protected.DELETE("/system-settings/:id", dep.SystemSettingHandler.Delete)
		protected.GET("/system/api-registry", func(c *gin.Context) {
			responseRegistry(c)
		})

		protected.GET("/projects", dep.ProjectHandler.ListProjects)
		protected.POST("/projects", dep.ProjectHandler.CreateProject)
		protected.PUT("/projects/:id", dep.ProjectHandler.UpdateProject)
		protected.DELETE("/projects/:id", dep.ProjectHandler.DeleteProject)
		protected.GET("/projects/:id/environments", dep.ProjectHandler.ListEnvironments)
		protected.POST("/projects/:id/environments", dep.ProjectHandler.CreateEnvironment)
		protected.PUT("/projects/:id/environments/:environmentId", dep.ProjectHandler.UpdateEnvironment)
		protected.DELETE("/projects/:id/environments/:environmentId", dep.ProjectHandler.DeleteEnvironment)
		protected.GET("/projects/stacks", dep.ProjectHandler.ListStacks)
		protected.POST("/projects/stacks", dep.ProjectHandler.CreateStack)
		protected.PUT("/projects/stacks/:id", dep.ProjectHandler.UpdateStack)
		protected.DELETE("/projects/stacks/:id", dep.ProjectHandler.DeleteStack)
		protected.GET("/graph/projects/:id", dep.GraphHandler.GetProjectGraph)

		protected.GET("/cloud/summary", dep.CloudHandler.Summary)
		protected.GET("/cloud/workbench", dep.CloudHandler.Workbench)
		protected.GET("/cloud/accounts", dep.CloudHandler.ListAccounts)
		protected.POST("/cloud/accounts", dep.CloudHandler.CreateAccount)
		protected.POST("/cloud/accounts/:id/test", dep.CloudHandler.TestAccount)
		protected.GET("/cloud/accounts/:id/instance-types", dep.CloudHandler.ListAccountInstanceTypes)
		protected.POST("/cloud/accounts/:id/key-pairs", dep.CloudHandler.CreateAccountKeyPair)
		protected.DELETE("/cloud/accounts/:id", dep.CloudHandler.DeleteAccount)
		protected.GET("/cloud/network-plans", dep.CloudHandler.ListNetworkPlans)
		protected.POST("/cloud/network-plans", dep.CloudHandler.CreateNetworkPlan)
		protected.PUT("/cloud/network-plans/:id", dep.CloudHandler.UpdateNetworkPlan)
		protected.DELETE("/cloud/network-plans/:id", dep.CloudHandler.DeleteNetworkPlan)
		protected.GET("/cloud/blueprints", dep.CloudHandler.ListBlueprints)
		protected.GET("/cloud/resources", dep.CloudHandler.ListResources)
		protected.POST("/cloud/resources/:id/retry-enrollment", dep.CloudHandler.RetryResourceEnrollment)
		protected.GET("/cloud/jobs", dep.CloudHandler.ListJobs)
		protected.GET("/cloud/jobs/:id", dep.CloudHandler.GetJob)
		protected.GET("/cloud/jobs/:id/logs", dep.CloudHandler.ListJobLogs)
		protected.POST("/cloud/jobs", dep.CloudHandler.CreateJob)
		protected.POST("/cloud/jobs/:id/cancel", dep.CloudHandler.CancelJob)
		protected.POST("/cloud/jobs/:id/retry", dep.CloudHandler.RetryJob)

		protected.GET("/k8s/clusters", dep.K8sHandler.ListClusters)
		protected.POST("/k8s/clusters", dep.K8sHandler.CreateCluster)
		protected.PUT("/k8s/clusters/:id", dep.K8sHandler.UpdateCluster)
		protected.DELETE("/k8s/clusters/:id", dep.K8sHandler.DeleteCluster)
		protected.GET("/k8s/clusters/:id/addons", dep.CloudHandler.ListClusterAddons)
		protected.POST("/k8s/clusters/:id/addons/:addonKey/retry", dep.CloudHandler.RetryClusterAddon)
		protected.POST("/k8s/clusters/:id/test", dep.K8sHandler.TestCluster)
		protected.GET("/k8s/clusters/:id/overview", dep.K8sHandler.GetClusterOverview)
		protected.POST("/k8s/clusters/:id/sync-namespaces", dep.K8sHandler.SyncNamespaces)
		protected.POST("/k8s/clusters/:id/sync-workloads", dep.K8sHandler.SyncWorkloads)
		protected.GET("/k8s/namespaces", dep.K8sHandler.ListNamespaces)
		protected.GET("/k8s/workloads", dep.K8sHandler.ListWorkloads)
		protected.GET("/k8s/workloads/:id", dep.K8sHandler.GetWorkload)
		protected.GET("/k8s/workloads/:id/rollout", dep.K8sHandler.GetWorkloadRollout)
		protected.GET("/k8s/workloads/:id/resources", dep.K8sHandler.GetWorkloadResources)
		protected.GET("/k8s/workloads/:id/services/:name", dep.K8sHandler.GetWorkloadServiceDetail)
		protected.GET("/k8s/workloads/:id/ingresses/:name", dep.K8sHandler.GetWorkloadIngressDetail)
		protected.GET("/k8s/workloads/:id/pods", dep.K8sHandler.GetWorkloadPods)
		protected.GET("/k8s/workloads/:id/events", dep.K8sHandler.GetWorkloadEvents)
		protected.GET("/k8s/workloads/:id/logs", dep.K8sHandler.GetWorkloadLogs)
		protected.POST("/k8s/workloads/:id/exec", dep.K8sHandler.ExecWorkloadCommand)
		protected.POST("/k8s/workloads/:id/shell-sessions", dep.K8sHandler.OpenShellSession)
		protected.POST("/k8s/workloads/:id/shell-sessions/:sessionId/input", dep.K8sHandler.SendShellInput)
		protected.GET("/k8s/workloads/:id/shell-sessions/:sessionId/output", dep.K8sHandler.ReadShellOutput)
		protected.DELETE("/k8s/workloads/:id/shell-sessions/:sessionId", dep.K8sHandler.CloseShellSession)
		protected.GET("/k8s/workloads/:id/manifest", dep.K8sHandler.GetWorkloadManifest)
		protected.GET("/k8s/workloads/:id/container-images", dep.K8sHandler.GetWorkloadContainerImages)
		protected.POST("/k8s/workloads/:id/scale", dep.K8sHandler.ScaleWorkload)
		protected.POST("/k8s/workloads/:id/image", dep.K8sHandler.UpdateWorkloadPrimaryImage)
		protected.POST("/k8s/workloads/:id/images", dep.K8sHandler.UpdateWorkloadImages)
		protected.GET("/k8s/workloads/:id/statefulset-settings", dep.K8sHandler.GetStatefulSetSettings)
		protected.POST("/k8s/workloads/:id/statefulset-settings", dep.K8sHandler.UpdateStatefulSetSettings)
		protected.POST("/k8s/workloads/:id/restart", dep.K8sHandler.RestartWorkload)
		protected.POST("/k8s/workloads/:id/rollout-restart", dep.K8sHandler.RolloutRestartWorkload)
		protected.DELETE("/k8s/workloads/:id", dep.K8sHandler.DeleteWorkload)
		protected.POST("/k8s/workloads/bundle", dep.K8sHandler.CreateWorkloadBundle)
		protected.POST("/k8s/manifests/apply", dep.K8sHandler.ApplyManifest)
		protected.GET("/k8s/resources", dep.K8sHandler.ListNamespaceResources)
		protected.GET("/k8s/resources/detail", dep.K8sHandler.GetNamespaceResourceDetail)
		protected.GET("/k8s/resources/manifest", dep.K8sHandler.GetNamespaceResourceManifest)
		protected.PUT("/k8s/resources/detail", dep.K8sHandler.UpdateNamespaceResource)
		protected.DELETE("/k8s/resources/detail", dep.K8sHandler.DeleteNamespaceResource)
		protected.POST("/k8s/namespaces", dep.K8sHandler.CreateNamespace)
		protected.PUT("/k8s/namespaces/:id", dep.K8sHandler.UpdateNamespace)
		protected.DELETE("/k8s/namespaces/:id", dep.K8sHandler.DeleteNamespace)

		protected.GET("/machines/summary", dep.MachineHandler.Summary)
		protected.GET("/machines/groups", dep.MachineHandler.ListGroups)
		protected.POST("/machines/groups", dep.MachineHandler.CreateGroup)
		protected.PUT("/machines/groups/:groupId", dep.MachineHandler.UpdateGroup)
		protected.DELETE("/machines/groups/:groupId", dep.MachineHandler.DeleteGroup)
		protected.GET("/machines/credentials", dep.MachineHandler.ListCredentials)
		protected.GET("/machines/credentials/:credentialId", dep.MachineHandler.GetCredential)
		protected.POST("/machines/credentials", dep.MachineHandler.CreateCredential)
		protected.DELETE("/machines/credentials/:credentialId", dep.MachineHandler.DeleteCredential)
		protected.GET("/machines/quick-commands", dep.MachineHandler.ListQuickCommands)
		protected.POST("/machines/quick-commands", dep.MachineHandler.CreateQuickCommand)
		protected.PUT("/machines/quick-commands/:commandId", dep.MachineHandler.UpdateQuickCommand)
		protected.DELETE("/machines/quick-commands/:commandId", dep.MachineHandler.DeleteQuickCommand)
		protected.GET("/machines/assets", dep.MachineHandler.ListAssets)
		protected.POST("/machines/assets", dep.MachineHandler.CreateAsset)
		protected.POST("/machines/assets/batch-create", dep.MachineHandler.BatchCreateAssets)
		protected.POST("/machines/assets/batch-update", dep.MachineHandler.BatchUpdateAssets)
		protected.POST("/machines/assets/batch-delete", dep.MachineHandler.BatchDeleteAssets)
		protected.PUT("/machines/assets/:id", dep.MachineHandler.UpdateAsset)
		protected.DELETE("/machines/assets/:id", dep.MachineHandler.DeleteAsset)
		protected.GET("/machines/assets/:id", dep.MachineHandler.GetAsset)
		protected.GET("/machines/assets/:id/sessions", dep.MachineHandler.ListAssetSessions)
		protected.GET("/machines/assets/:id/events", dep.MachineHandler.ListAssetEvents)
		protected.GET("/machines/assets/:id/sftp", dep.MachineHandler.ListAssetSFTP)
		protected.GET("/machines/assets/:id/sftp/download", dep.MachineHandler.DownloadAssetSFTP)
		protected.POST("/machines/assets/:id/sftp/upload", dep.MachineHandler.UploadAssetSFTP)
		protected.GET("/machines/assets/:id/accounts", dep.MachineHandler.ListAccounts)
		protected.GET("/machines/assets/:id/accounts/:accountId", dep.MachineHandler.GetAccount)
		protected.POST("/machines/assets/:id/accounts", dep.MachineHandler.CreateAccount)
		protected.POST("/machines/assets/:id/accounts/import", dep.MachineHandler.ImportCredential)
		protected.POST("/machines/assets/:id/accounts/:accountId/default", dep.MachineHandler.SetDefaultAccount)
		protected.DELETE("/machines/assets/:id/accounts/:accountId", dep.MachineHandler.DeleteAccount)
		protected.POST("/machines/assets/:id/terminal-ticket", dep.MachineHandler.CreateTerminalTicket)
		protected.GET("/machines/sessions", dep.MachineHandler.ListSessions)
		protected.GET("/machines/events", dep.MachineHandler.ListRecentEvents)
		protected.POST("/machines/quick-connect", dep.MachineHandler.QuickConnect)
	}

	internalCloud := api.Group("/cloud/internal")
	internalCloud.Use(middleware.StaticBearer(dep.Config.InternalRunnerToken))
	{
		internalCloud.POST("/jobs/claim", dep.CloudHandler.ClaimNextJob)
		internalCloud.POST("/jobs/:id/heartbeat", dep.CloudHandler.HeartbeatJob)
		internalCloud.POST("/jobs/:id/logs", dep.CloudHandler.AppendJobLog)
		internalCloud.POST("/jobs/:id/result", dep.CloudHandler.ReportJobResult)
		internalCloud.POST("/resource-sync/claim", dep.CloudHandler.ClaimNextResourceSync)
		internalCloud.POST("/resource-sync/:id/result", dep.CloudHandler.ReportResourceSyncResult)
		internalCloud.POST("/machine-enrollment/claim", dep.CloudHandler.ClaimNextMachineEnrollment)
		internalCloud.POST("/machine-enrollment/:id/result", dep.CloudHandler.ReportMachineEnrollmentResult)
		internalCloud.POST("/cluster-enrollment/claim", dep.CloudHandler.ClaimNextClusterEnrollment)
		internalCloud.POST("/cluster-enrollment/:id/result", dep.CloudHandler.ReportClusterEnrollmentResult)
		internalCloud.POST("/cluster-addons/claim", dep.CloudHandler.ClaimNextClusterAddon)
		internalCloud.POST("/cluster-addons/:id/heartbeat", dep.CloudHandler.HeartbeatClusterAddon)
		internalCloud.POST("/cluster-addons/:id/result", dep.CloudHandler.ReportClusterAddonResult)
	}

	api.GET("/machines/assets/:id/terminal", dep.MachineHandler.OpenInteractiveTerminal)

	return router
}
