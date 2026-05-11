package project

import (
	"net/http"
	"strconv"

	"backend-center/internal/infra/web/response"

	"github.com/gin-gonic/gin"
)

func parseProjectPathID(c *gin.Context) (uint, error) {
	raw := c.Param("projectId")
	if raw == "" {
		raw = c.Param("id")
	}
	id, err := strconv.Atoi(raw)
	if err != nil {
		return 0, err
	}
	return uint(id), nil
}

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) ListProjects(c *gin.Context) {
	items, err := h.service.ListProjects()
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(c, http.StatusOK, items)
}

func (h *Handler) CreateProject(c *gin.Context) {
	var input ProjectInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.service.CreateProject(input)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(c, http.StatusCreated, item)
}

func (h *Handler) UpdateProject(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid project id")
		return
	}
	var input ProjectInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.service.UpdateProject(uint(id), input)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(c, http.StatusOK, item)
}

func (h *Handler) DeleteProject(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid project id")
		return
	}
	if err := h.service.DeleteProject(uint(id)); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(c, http.StatusOK, gin.H{"deleted": true})
}

func (h *Handler) ListEnvironments(c *gin.Context) {
	projectID, err := parseProjectPathID(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid project id")
		return
	}
	items, err := h.service.ListEnvironments(projectID)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(c, http.StatusOK, items)
}

func (h *Handler) CreateEnvironment(c *gin.Context) {
	projectID, err := parseProjectPathID(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid project id")
		return
	}
	var input EnvironmentInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.service.CreateEnvironment(projectID, input)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(c, http.StatusCreated, item)
}

func (h *Handler) UpdateEnvironment(c *gin.Context) {
	projectID, err := parseProjectPathID(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid project id")
		return
	}
	environmentID, err := strconv.Atoi(c.Param("environmentId"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid environment id")
		return
	}
	var input EnvironmentInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.service.UpdateEnvironment(projectID, uint(environmentID), input)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(c, http.StatusOK, item)
}

func (h *Handler) DeleteEnvironment(c *gin.Context) {
	projectID, err := parseProjectPathID(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid project id")
		return
	}
	environmentID, err := strconv.Atoi(c.Param("environmentId"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid environment id")
		return
	}
	if err := h.service.DeleteEnvironment(projectID, uint(environmentID)); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(c, http.StatusOK, gin.H{"deleted": true})
}

func (h *Handler) ListStacks(c *gin.Context) {
	var projectID *uint
	if raw := c.Query("project_id"); raw != "" {
		id, err := strconv.Atoi(raw)
		if err != nil {
			response.Error(c, http.StatusBadRequest, "invalid project id")
			return
		}
		value := uint(id)
		projectID = &value
	}
	var environmentID *uint
	if raw := c.Query("environment_id"); raw != "" {
		id, err := strconv.Atoi(raw)
		if err != nil {
			response.Error(c, http.StatusBadRequest, "invalid environment id")
			return
		}
		value := uint(id)
		environmentID = &value
	}
	items, err := h.service.ListStacks(projectID, environmentID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(c, http.StatusOK, items)
}

func (h *Handler) CreateStack(c *gin.Context) {
	var input StackInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.service.CreateStack(input)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(c, http.StatusCreated, item)
}

func (h *Handler) UpdateStack(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid stack id")
		return
	}
	var input StackInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.service.UpdateStack(uint(id), input)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(c, http.StatusOK, item)
}

func (h *Handler) DeleteStack(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid stack id")
		return
	}
	if err := h.service.DeleteStack(uint(id)); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(c, http.StatusOK, gin.H{"deleted": true})
}
