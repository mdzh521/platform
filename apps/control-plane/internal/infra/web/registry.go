package http

import (
	nethttp "net/http"

	"github.com/gin-gonic/gin"
)

type APIRegistryEntry struct {
	Domain      string `json:"domain"`
	Method      string `json:"method"`
	Path        string `json:"path"`
	Description string `json:"description"`
	AuthMode    string `json:"auth_mode"`
	Version     string `json:"version"`
}

var apiRegistryEntries = []APIRegistryEntry{
	{Domain: "project", Method: "GET", Path: "/api/v1/projects", Description: "List projects", AuthMode: "jwt", Version: "v1"},
	{Domain: "project", Method: "POST", Path: "/api/v1/projects", Description: "Create project", AuthMode: "jwt", Version: "v1"},
	{Domain: "project", Method: "GET", Path: "/api/v1/projects/:projectId/environments", Description: "List environments by project", AuthMode: "jwt", Version: "v1"},
	{Domain: "project", Method: "GET", Path: "/api/v1/projects/stacks", Description: "List stacks", AuthMode: "jwt", Version: "v1"},
	{Domain: "cloud", Method: "GET", Path: "/api/v1/cloud/accounts", Description: "List cloud accounts", AuthMode: "jwt", Version: "v1"},
	{Domain: "cloud", Method: "GET", Path: "/api/v1/cloud/network-plans", Description: "List foundation networks", AuthMode: "jwt", Version: "v1"},
	{Domain: "cloud", Method: "POST", Path: "/api/v1/cloud/jobs", Description: "Create deployment job", AuthMode: "jwt", Version: "v1"},
	{Domain: "cloud", Method: "GET", Path: "/api/v1/cloud/resources", Description: "List cloud resources", AuthMode: "jwt", Version: "v1"},
	{Domain: "cloud", Method: "POST", Path: "/api/v1/cloud/resources/:id/retry-enrollment", Description: "Retry cloud resource enrollment", AuthMode: "jwt", Version: "v1"},
	{Domain: "k8s", Method: "GET", Path: "/api/v1/k8s/clusters/:id/addons", Description: "List cluster addon executions", AuthMode: "jwt", Version: "v1"},
	{Domain: "k8s", Method: "POST", Path: "/api/v1/k8s/clusters/:id/addons/:addonKey/retry", Description: "Retry cluster addon execution", AuthMode: "jwt", Version: "v1"},
	{Domain: "machine", Method: "GET", Path: "/api/v1/machines/assets", Description: "List machine assets", AuthMode: "jwt", Version: "v1"},
	{Domain: "k8s", Method: "GET", Path: "/api/v1/k8s/clusters", Description: "List managed clusters", AuthMode: "jwt", Version: "v1"},
	{Domain: "graph", Method: "GET", Path: "/api/v1/graph/projects/:id", Description: "Query project graph", AuthMode: "jwt", Version: "v1"},
	{Domain: "system", Method: "GET", Path: "/api/v1/system/api-registry", Description: "List API registry", AuthMode: "jwt", Version: "v1"},
	{Domain: "internal", Method: "POST", Path: "/api/v1/cloud/internal/resource-sync/claim", Description: "Claim resource sync work", AuthMode: "static_bearer", Version: "v1"},
	{Domain: "internal", Method: "POST", Path: "/api/v1/cloud/internal/machine-enrollment/claim", Description: "Claim machine enrollment work", AuthMode: "static_bearer", Version: "v1"},
	{Domain: "internal", Method: "POST", Path: "/api/v1/cloud/internal/cluster-enrollment/claim", Description: "Claim cluster enrollment work", AuthMode: "static_bearer", Version: "v1"},
	{Domain: "internal", Method: "POST", Path: "/api/v1/cloud/internal/cluster-addons/claim", Description: "Claim cluster addon work", AuthMode: "static_bearer", Version: "v1"},
}

func responseRegistry(c *gin.Context) {
	c.JSON(nethttp.StatusOK, gin.H{
		"data": apiRegistryEntries,
	})
}
