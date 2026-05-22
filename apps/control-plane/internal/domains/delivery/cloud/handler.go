package cloud

import (
	"net/http"
	"strconv"

	"backend-center/internal/infra/web/response"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Summary(c *gin.Context) {
	data, err := h.service.Summary.Summary()
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(c, http.StatusOK, data)
}

func (h *Handler) Workbench(c *gin.Context) {
	data, err := h.service.Workbench.Get()
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(c, http.StatusOK, data)
}

func (h *Handler) ListAccounts(c *gin.Context) {
	data, err := h.service.Accounts.List()
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(c, http.StatusOK, data)
}

func (h *Handler) CreateAccount(c *gin.Context) {
	var input CloudAccountInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	data, err := h.service.Accounts.Create(input)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(c, http.StatusOK, data)
}

func (h *Handler) TestAccount(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid account id")
		return
	}
	data, err := h.service.Accounts.Test(uint(id))
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(c, http.StatusOK, data)
}

func (h *Handler) ListAccountInstanceTypes(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid account id")
		return
	}
	data, err := h.service.Accounts.ListInstanceTypesWithZone(uint(id), c.Query("provider"), c.Query("region"), c.Query("zone"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(c, http.StatusOK, data)
}

func (h *Handler) CreateAccountKeyPair(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid account id")
		return
	}
	var input CloudKeyPairInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	data, err := h.service.Accounts.CreateKeyPair(uint(id), input)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(c, http.StatusOK, data)
}

func (h *Handler) DeleteAccount(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid account id")
		return
	}
	if err := h.service.Accounts.Delete(uint(id)); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(c, http.StatusOK, gin.H{"deleted": true})
}

func (h *Handler) ListNetworkPlans(c *gin.Context) {
	data, err := h.service.Networks.List()
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(c, http.StatusOK, data)
}

func (h *Handler) CreateNetworkPlan(c *gin.Context) {
	var input NetworkPlanInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	data, err := h.service.Networks.Create(input)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(c, http.StatusOK, data)
}

func (h *Handler) UpdateNetworkPlan(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid network plan id")
		return
	}
	var input NetworkPlanInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	data, err := h.service.Networks.Update(uint(id), input)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(c, http.StatusOK, data)
}

func (h *Handler) DeleteNetworkPlan(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid network plan id")
		return
	}
	if err := h.service.Networks.Delete(uint(id)); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(c, http.StatusOK, gin.H{"deleted": true})
}

func (h *Handler) ListBlueprints(c *gin.Context) {
	data, err := h.service.Blueprints.List()
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(c, http.StatusOK, data)
}

func (h *Handler) ListResources(c *gin.Context) {
	data, err := h.service.Resources.List()
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(c, http.StatusOK, data)
}

func (h *Handler) RetryResourceEnrollment(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid resource id")
		return
	}
	var input RetryResourceEnrollmentInput
	_ = c.ShouldBindJSON(&input)
	if err := h.service.Resources.RetryResourceEnrollment(uint(id), input); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(c, http.StatusOK, gin.H{"ack": true})
}

func (h *Handler) ListClusterAddons(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid cluster id")
		return
	}
	data, err := h.service.Addons.ListByCluster(uint(id))
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(c, http.StatusOK, data)
}

func (h *Handler) RetryClusterAddon(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid cluster id")
		return
	}
	if err := h.service.Addons.Retry(uint(id), c.Param("addonKey")); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(c, http.StatusOK, gin.H{"ack": true})
}

func (h *Handler) ListJobs(c *gin.Context) {
	data, err := h.service.Deployments.List()
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(c, http.StatusOK, data)
}

func (h *Handler) GetJob(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid job id")
		return
	}
	data, err := h.service.Deployments.Get(uint(id))
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(c, http.StatusOK, data)
}

func (h *Handler) ListJobLogs(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid job id")
		return
	}
	data, err := h.service.Deployments.ListLogs(uint(id))
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(c, http.StatusOK, data)
}

func (h *Handler) CreateJob(c *gin.Context) {
	var input DeploymentJobInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	data, err := h.service.Deployments.Create(input)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(c, http.StatusOK, data)
}

func (h *Handler) CancelJob(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid job id")
		return
	}
	data, err := h.service.Deployments.Cancel(uint(id))
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(c, http.StatusOK, data)
}

func (h *Handler) RetryJob(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid job id")
		return
	}
	var input RetryJobInput
	_ = c.ShouldBindJSON(&input)
	data, err := h.service.Deployments.Retry(uint(id), input)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(c, http.StatusOK, data)
}

func (h *Handler) ClaimNextJob(c *gin.Context) {
	var input JobClaimRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	data, err := h.service.Deployments.ClaimNext(c.Request.Context(), input)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(c, http.StatusOK, data)
}

func (h *Handler) HeartbeatJob(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid job id")
		return
	}
	var input JobHeartbeatInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.service.Deployments.Heartbeat(uint(id), input); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(c, http.StatusOK, gin.H{"ack": true})
}

func (h *Handler) AppendJobLog(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid job id")
		return
	}
	var input JobLogInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.service.Deployments.AppendLog(uint(id), input); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(c, http.StatusOK, gin.H{"ack": true})
}

func (h *Handler) ReportJobResult(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid job id")
		return
	}
	var input JobResultInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	data, err := h.service.Deployments.ReportResult(uint(id), input)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(c, http.StatusOK, data)
}

func (h *Handler) ClaimNextResourceSync(c *gin.Context) {
	var input ResourceSyncClaimRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	data, err := h.service.Resources.ClaimNextSync(input)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(c, http.StatusOK, data)
}

func (h *Handler) ReportResourceSyncResult(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid job id")
		return
	}
	var input ResourceSyncResultInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.service.Resources.ReportSyncResult(uint(id), input); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(c, http.StatusOK, gin.H{"ack": true})
}

func (h *Handler) ClaimNextMachineEnrollment(c *gin.Context) {
	var input MachineEnrollmentClaimRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	data, err := h.service.Resources.ClaimNextMachineEnrollment(c.Request.Context(), input)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(c, http.StatusOK, data)
}

func (h *Handler) ReportMachineEnrollmentResult(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid resource id")
		return
	}
	var input MachineEnrollmentResultInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.service.Resources.ReportMachineEnrollmentResult(uint(id), input); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(c, http.StatusOK, gin.H{"ack": true})
}

func (h *Handler) ClaimNextClusterEnrollment(c *gin.Context) {
	var input ClusterEnrollmentClaimRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	data, err := h.service.Resources.ClaimNextClusterEnrollment(c.Request.Context(), input)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(c, http.StatusOK, data)
}

func (h *Handler) ReportClusterEnrollmentResult(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid resource id")
		return
	}
	var input ClusterEnrollmentResultInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.service.Resources.ReportClusterEnrollmentResult(uint(id), input); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(c, http.StatusOK, gin.H{"ack": true})
}

func (h *Handler) ClaimNextClusterAddon(c *gin.Context) {
	var input ClusterAddonClaimRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	data, err := h.service.Addons.ClaimNext(c.Request.Context(), input)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(c, http.StatusOK, data)
}

func (h *Handler) HeartbeatClusterAddon(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid addon execution id")
		return
	}
	var input ClusterAddonHeartbeatInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.service.Addons.Heartbeat(uint(id), input); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(c, http.StatusOK, gin.H{"ack": true})
}

func (h *Handler) ReportClusterAddonResult(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid addon execution id")
		return
	}
	var input ClusterAddonResultInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.service.Addons.ReportResult(uint(id), input); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(c, http.StatusOK, gin.H{"ack": true})
}
