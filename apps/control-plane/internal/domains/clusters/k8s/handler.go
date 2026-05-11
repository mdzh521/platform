package k8s

import (
	"net/http"
	"strconv"

	"backend-center/internal/infra/web/response"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Handler groups K8s HTTP endpoints.
// Keep generic route wiring and simple request/response conversion here.
// Terminal-specific streaming endpoints should remain in handler_terminal.go.
type Handler struct {
	service        *Service
	jwtSecret      string
	trustedOrigins []string
}

func NewHandler(service *Service, jwtSecret string, trustedOrigins []string) *Handler {
	return &Handler{service: service, jwtSecret: jwtSecret, trustedOrigins: trustedOrigins}
}

func writeServiceError(c *gin.Context, err error) {
	switch {
	case err == nil:
		return
	case err == gorm.ErrRecordNotFound:
		response.Error(c, http.StatusNotFound, "resource not found")
	default:
		response.Error(c, http.StatusBadRequest, err.Error())
	}
}

func (h *Handler) ListClusters(c *gin.Context) {
	items, err := h.service.ListClusters()
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(c, http.StatusOK, items)
}

func (h *Handler) CreateCluster(c *gin.Context) {
	var input ClusterInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.service.CreateCluster(input)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(c, http.StatusCreated, item)
}

func (h *Handler) UpdateCluster(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid cluster id")
		return
	}
	var input ClusterInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.service.UpdateCluster(uint(id), input)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(c, http.StatusOK, item)
}

func (h *Handler) DeleteCluster(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid cluster id")
		return
	}
	if err := h.service.DeleteCluster(uint(id)); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(c, http.StatusOK, gin.H{"deleted": true})
}

func (h *Handler) TestCluster(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid cluster id")
		return
	}
	item, err := h.service.TestCluster(uint(id))
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(c, http.StatusOK, item)
}

func (h *Handler) GetClusterOverview(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid cluster id")
		return
	}
	item, err := h.service.GetClusterOverview(uint(id))
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(c, http.StatusOK, item)
}

func (h *Handler) SyncNamespaces(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid cluster id")
		return
	}
	item, err := h.service.SyncNamespaces(uint(id))
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(c, http.StatusOK, item)
}

func (h *Handler) SyncWorkloads(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid cluster id")
		return
	}
	item, err := h.service.SyncWorkloads(uint(id))
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(c, http.StatusOK, item)
}

func (h *Handler) ListNamespaces(c *gin.Context) {
	var clusterID *uint
	if raw := c.Query("cluster_id"); raw != "" {
		id, err := strconv.Atoi(raw)
		if err != nil {
			response.Error(c, http.StatusBadRequest, "invalid cluster id")
			return
		}
		value := uint(id)
		clusterID = &value
	}
	items, err := h.service.ListNamespaces(clusterID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(c, http.StatusOK, items)
}

func (h *Handler) ListWorkloads(c *gin.Context) {
	var clusterID *uint
	if raw := c.Query("cluster_id"); raw != "" {
		id, err := strconv.Atoi(raw)
		if err != nil {
			response.Error(c, http.StatusBadRequest, "invalid cluster id")
			return
		}
		value := uint(id)
		clusterID = &value
	}
	includePods := c.Query("include_pods") == "true"
	items, err := h.service.ListWorkloads(clusterID, c.Query("namespace"), includePods)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(c, http.StatusOK, items)
}

func (h *Handler) GetWorkload(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid workload id")
		return
	}
	item, err := h.service.GetWorkload(uint(id))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	response.JSON(c, http.StatusOK, item)
}

func (h *Handler) GetWorkloadEvents(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid workload id")
		return
	}
	item, err := h.service.GetWorkloadEvents(uint(id))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	response.JSON(c, http.StatusOK, item)
}

func (h *Handler) GetWorkloadRollout(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid workload id")
		return
	}
	item, err := h.service.GetWorkloadRollout(uint(id))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	response.JSON(c, http.StatusOK, item)
}

func (h *Handler) GetWorkloadResources(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid workload id")
		return
	}
	item, err := h.service.GetWorkloadResources(uint(id))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	response.JSON(c, http.StatusOK, item)
}

func (h *Handler) GetWorkloadServiceDetail(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid workload id")
		return
	}
	item, err := h.service.GetWorkloadServiceDetail(uint(id), c.Param("name"))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	response.JSON(c, http.StatusOK, item)
}

func (h *Handler) GetWorkloadIngressDetail(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid workload id")
		return
	}
	item, err := h.service.GetWorkloadIngressDetail(uint(id), c.Param("name"))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	response.JSON(c, http.StatusOK, item)
}

func (h *Handler) GetWorkloadPods(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid workload id")
		return
	}
	item, err := h.service.GetWorkloadPods(uint(id))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	response.JSON(c, http.StatusOK, item)
}

func (h *Handler) GetWorkloadLogs(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid workload id")
		return
	}
	tailLines := 200
	if raw := c.Query("tail_lines"); raw != "" {
		if value, err := strconv.Atoi(raw); err == nil && value > 0 {
			tailLines = value
		}
	}
	item, err := h.service.GetWorkloadLogs(uint(id), tailLines, c.Query("pod_name"))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	response.JSON(c, http.StatusOK, item)
}

func (h *Handler) ExecWorkloadCommand(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid workload id")
		return
	}
	var input ExecWorkloadCommandInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.service.ExecWorkloadCommand(uint(id), input)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	response.JSON(c, http.StatusOK, item)
}

func (h *Handler) OpenShellSession(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid workload id")
		return
	}
	var input OpenShellSessionInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.service.OpenShellSession(uint(id), input)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	response.JSON(c, http.StatusOK, item)
}

func (h *Handler) SendShellInput(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid workload id")
		return
	}
	var input ShellSessionInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.service.SendShellInput(uint(id), c.Param("sessionId"), input.Input)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	response.JSON(c, http.StatusOK, item)
}

func (h *Handler) ReadShellOutput(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid workload id")
		return
	}
	offset := 0
	if raw := c.Query("offset"); raw != "" {
		value, convErr := strconv.Atoi(raw)
		if convErr != nil {
			response.Error(c, http.StatusBadRequest, "invalid offset")
			return
		}
		offset = value
	}
	item, err := h.service.ReadShellOutput(uint(id), c.Param("sessionId"), offset)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	response.JSON(c, http.StatusOK, item)
}

func (h *Handler) CloseShellSession(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid workload id")
		return
	}
	if err := h.service.CloseShellSession(uint(id), c.Param("sessionId")); err != nil {
		writeServiceError(c, err)
		return
	}
	response.JSON(c, http.StatusOK, gin.H{"deleted": true})
}

func (h *Handler) GetWorkloadManifest(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid workload id")
		return
	}
	mode := c.DefaultQuery("mode", "compact")
	if mode != "compact" && mode != "full" {
		response.Error(c, http.StatusBadRequest, "invalid manifest mode")
		return
	}
	item, err := h.service.GetWorkloadManifest(uint(id), mode)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	response.JSON(c, http.StatusOK, item)
}

func (h *Handler) ScaleWorkload(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid workload id")
		return
	}
	var input ScaleWorkloadInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.service.ScaleWorkload(uint(id), input.Replicas)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(c, http.StatusOK, item)
}

func (h *Handler) UpdateWorkloadPrimaryImage(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid workload id")
		return
	}
	var input UpdatePrimaryImageInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.service.UpdateWorkloadPrimaryImage(uint(id), input.Image)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	response.JSON(c, http.StatusOK, item)
}

func (h *Handler) GetWorkloadContainerImages(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid workload id")
		return
	}
	item, err := h.service.GetWorkloadContainerImages(uint(id))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	response.JSON(c, http.StatusOK, item)
}

func (h *Handler) UpdateWorkloadImages(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid workload id")
		return
	}
	var input UpdateWorkloadImagesInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.service.UpdateWorkloadImages(uint(id), input)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	response.JSON(c, http.StatusOK, item)
}

func (h *Handler) GetStatefulSetSettings(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid workload id")
		return
	}
	item, err := h.service.GetStatefulSetSettings(uint(id))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	response.JSON(c, http.StatusOK, item)
}

func (h *Handler) UpdateStatefulSetSettings(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid workload id")
		return
	}
	var input UpdateStatefulSetSettingsInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.service.UpdateStatefulSetSettings(uint(id), input)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	response.JSON(c, http.StatusOK, item)
}

func (h *Handler) RestartWorkload(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid workload id")
		return
	}
	item, err := h.service.RestartWorkload(uint(id))
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(c, http.StatusOK, item)
}

func (h *Handler) RolloutRestartWorkload(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid workload id")
		return
	}
	item, err := h.service.RolloutRestartWorkload(uint(id))
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(c, http.StatusOK, item)
}

func (h *Handler) DeleteWorkload(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid workload id")
		return
	}
	item, err := h.service.DeleteWorkload(uint(id))
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(c, http.StatusOK, item)
}

func (h *Handler) CreateWorkloadBundle(c *gin.Context) {
	var input CreateWorkloadBundleInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.service.CreateWorkloadBundle(input)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(c, http.StatusCreated, item)
}

func (h *Handler) ApplyManifest(c *gin.Context) {
	var input ApplyManifestInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.service.ApplyManifest(input)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(c, http.StatusCreated, item)
}

func (h *Handler) ListNamespaceResources(c *gin.Context) {
	clusterID, err := strconv.Atoi(c.Query("cluster_id"))
	if err != nil || clusterID <= 0 {
		response.Error(c, http.StatusBadRequest, "invalid cluster id")
		return
	}
	namespace := c.Query("namespace")
	kind := c.Query("kind")
	if namespace == "" || kind == "" {
		response.Error(c, http.StatusBadRequest, "namespace and kind are required")
		return
	}
	item, err := h.service.ListNamespaceResources(uint(clusterID), namespace, kind)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(c, http.StatusOK, item)
}

func (h *Handler) GetNamespaceResourceDetail(c *gin.Context) {
	clusterID, err := strconv.Atoi(c.Query("cluster_id"))
	if err != nil || clusterID <= 0 {
		response.Error(c, http.StatusBadRequest, "invalid cluster id")
		return
	}
	namespace := c.Query("namespace")
	kind := c.Query("kind")
	name := c.Query("name")
	if namespace == "" || kind == "" || name == "" {
		response.Error(c, http.StatusBadRequest, "cluster_id, namespace, kind and name are required")
		return
	}
	item, err := h.service.GetNamespaceResourceDetail(uint(clusterID), namespace, kind, name)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(c, http.StatusOK, item)
}

func (h *Handler) GetNamespaceResourceManifest(c *gin.Context) {
	clusterID, err := strconv.Atoi(c.Query("cluster_id"))
	if err != nil || clusterID <= 0 {
		response.Error(c, http.StatusBadRequest, "invalid cluster id")
		return
	}
	namespace := c.Query("namespace")
	kind := c.Query("kind")
	name := c.Query("name")
	mode := c.DefaultQuery("mode", "compact")
	if mode != "compact" && mode != "full" {
		response.Error(c, http.StatusBadRequest, "invalid manifest mode")
		return
	}
	if namespace == "" || kind == "" || name == "" {
		response.Error(c, http.StatusBadRequest, "cluster_id, namespace, kind and name are required")
		return
	}
	item, err := h.service.GetNamespaceResourceManifest(uint(clusterID), namespace, kind, name, mode)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(c, http.StatusOK, item)
}

func (h *Handler) DeleteNamespaceResource(c *gin.Context) {
	clusterID, err := strconv.Atoi(c.Query("cluster_id"))
	if err != nil || clusterID <= 0 {
		response.Error(c, http.StatusBadRequest, "invalid cluster id")
		return
	}
	namespace := c.Query("namespace")
	kind := c.Query("kind")
	name := c.Query("name")
	if namespace == "" || kind == "" || name == "" {
		response.Error(c, http.StatusBadRequest, "cluster_id, namespace, kind and name are required")
		return
	}
	item, err := h.service.DeleteNamespaceResource(uint(clusterID), namespace, kind, name)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(c, http.StatusOK, item)
}

func (h *Handler) UpdateNamespaceResource(c *gin.Context) {
	clusterID, err := strconv.Atoi(c.Query("cluster_id"))
	if err != nil || clusterID <= 0 {
		response.Error(c, http.StatusBadRequest, "invalid cluster id")
		return
	}
	namespace := c.Query("namespace")
	kind := c.Query("kind")
	name := c.Query("name")
	if namespace == "" || kind == "" || name == "" {
		response.Error(c, http.StatusBadRequest, "namespace, kind and name are required")
		return
	}
	var input UpdateNamespaceResourceInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.service.UpdateNamespaceResource(uint(clusterID), namespace, kind, name, input)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	response.JSON(c, http.StatusOK, item)
}

func (h *Handler) CreateNamespace(c *gin.Context) {
	var input NamespaceInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.service.CreateNamespace(input)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(c, http.StatusCreated, item)
}

func (h *Handler) UpdateNamespace(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid namespace id")
		return
	}
	var input NamespaceInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.service.UpdateNamespace(uint(id), input)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(c, http.StatusOK, item)
}

func (h *Handler) DeleteNamespace(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid namespace id")
		return
	}
	if err := h.service.DeleteNamespace(uint(id)); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(c, http.StatusOK, gin.H{"deleted": true})
}
