package machine

import (
	nethttp "net/http"
	"strconv"

	"backend-center/internal/domains/iam/auth"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service        *Service
	jwtSecret      string
	trustedOrigins []string
}

func NewHandler(service *Service, jwtSecret string, trustedOrigins []string) *Handler {
	return &Handler{service: service, jwtSecret: jwtSecret, trustedOrigins: trustedOrigins}
}

func currentClaims(c *gin.Context) (*auth.Claims, bool) {
	value, ok := c.Get("claims")
	if !ok {
		return nil, false
	}
	claims, ok := value.(*auth.Claims)
	return claims, ok
}

func isAdminViewer(c *gin.Context) bool {
	claims, ok := currentClaims(c)
	if !ok || claims == nil {
		return false
	}
	return claims.Username == "admin"
}

func (h *Handler) Summary(c *gin.Context) {
	data, err := h.service.Summary()
	if err != nil {
		c.JSON(nethttp.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(nethttp.StatusOK, gin.H{"data": data})
}

func (h *Handler) ListGroups(c *gin.Context) {
	data, err := h.service.ListGroups()
	if err != nil {
		c.JSON(nethttp.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(nethttp.StatusOK, gin.H{"data": data})
}

func (h *Handler) ListAssets(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	input := AssetListInput{
		Search:   c.Query("search"),
		Group:    c.Query("group"),
		Status:   c.Query("status"),
		Page:     page,
		PageSize: pageSize,
		SortBy:   c.DefaultQuery("sort_by", "updated_at"),
		Order:    c.DefaultQuery("order", "desc"),
	}
	data, err := h.service.ListAssets(input)
	if err != nil {
		c.JSON(nethttp.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(nethttp.StatusOK, gin.H{"data": data})
}

func (h *Handler) ListCredentials(c *gin.Context) {
	data, err := h.service.ListCredentials()
	if err != nil {
		c.JSON(nethttp.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(nethttp.StatusOK, gin.H{"data": data})
}

func (h *Handler) ListQuickCommands(c *gin.Context) {
	data, err := h.service.ListQuickCommands()
	if err != nil {
		c.JSON(nethttp.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(nethttp.StatusOK, gin.H{"data": data})
}

func (h *Handler) GetCredential(c *gin.Context) {
	if !isAdminViewer(c) {
		c.JSON(nethttp.StatusForbidden, gin.H{"error": "only admin can view credential material"})
		return
	}
	id, err := strconv.Atoi(c.Param("credentialId"))
	if err != nil {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": "invalid credential id"})
		return
	}
	data, err := h.service.GetCredential(uint(id))
	if err != nil {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(nethttp.StatusOK, gin.H{"data": data})
}

func (h *Handler) ListSessions(c *gin.Context) {
	data, err := h.service.ListSessions()
	if err != nil {
		c.JSON(nethttp.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(nethttp.StatusOK, gin.H{"data": data})
}

func (h *Handler) ListRecentEvents(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "40"))
	data, err := h.service.ListRecentEvents(limit)
	if err != nil {
		c.JSON(nethttp.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(nethttp.StatusOK, gin.H{"data": data})
}

func (h *Handler) GetAsset(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": "invalid asset id"})
		return
	}
	data, err := h.service.GetAsset(uint(id))
	if err != nil {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(nethttp.StatusOK, gin.H{"data": data})
}

func (h *Handler) ListAssetSessions(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": "invalid asset id"})
		return
	}
	data, err := h.service.ListAssetSessions(uint(id))
	if err != nil {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(nethttp.StatusOK, gin.H{"data": data})
}

func (h *Handler) ListAssetEvents(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": "invalid asset id"})
		return
	}
	data, err := h.service.ListAssetEvents(uint(id))
	if err != nil {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(nethttp.StatusOK, gin.H{"data": data})
}

func (h *Handler) ListAssetSFTP(c *gin.Context) {
	assetID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": "invalid asset id"})
		return
	}
	sessionID, err := strconv.Atoi(c.Query("session_id"))
	if err != nil || sessionID <= 0 {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": "invalid session id"})
		return
	}
	data, err := h.service.ListSFTP(uint(assetID), uint(sessionID), c.DefaultQuery("path", "."))
	if err != nil {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(nethttp.StatusOK, gin.H{"data": data})
}

func (h *Handler) DownloadAssetSFTP(c *gin.Context) {
	assetID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": "invalid asset id"})
		return
	}
	sessionID, err := strconv.Atoi(c.Query("session_id"))
	if err != nil || sessionID <= 0 {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": "invalid session id"})
		return
	}
	stream, filename, err := h.service.DownloadSFTP(uint(assetID), uint(sessionID), c.Query("path"))
	if err != nil {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	defer stream.Close()
	c.Header("Content-Disposition", `attachment; filename="`+filename+`"`)
	c.DataFromReader(nethttp.StatusOK, -1, "application/octet-stream", stream, nil)
}

func (h *Handler) UploadAssetSFTP(c *gin.Context) {
	assetID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": "invalid asset id"})
		return
	}
	sessionID, err := strconv.Atoi(c.PostForm("session_id"))
	if err != nil || sessionID <= 0 {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": "invalid session id"})
		return
	}
	targetPath := c.DefaultPostForm("path", "/")
	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": "missing upload file"})
		return
	}
	file, err := fileHeader.Open()
	if err != nil {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	defer file.Close()
	if err := h.service.UploadSFTP(uint(assetID), uint(sessionID), targetPath, fileHeader.Filename, file); err != nil {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(nethttp.StatusOK, gin.H{"data": gin.H{"message": "文件已上传"}})
}

func (h *Handler) ListAccounts(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": "invalid asset id"})
		return
	}
	data, err := h.service.ListAccounts(uint(id))
	if err != nil {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(nethttp.StatusOK, gin.H{"data": data})
}

func (h *Handler) GetAccount(c *gin.Context) {
	claims, ok := currentClaims(c)
	if !ok || claims == nil || claims.Username != "admin" {
		c.JSON(nethttp.StatusForbidden, gin.H{"error": "only admin can view credential material"})
		return
	}
	assetID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": "invalid asset id"})
		return
	}
	accountID, err := strconv.Atoi(c.Param("accountId"))
	if err != nil {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": "invalid account id"})
		return
	}
	data, err := h.service.GetAccount(uint(assetID), uint(accountID))
	if err != nil {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	h.service.recordSessionEvent(uint(assetID), 0, "credential_viewed", "warning", "查看登录凭据明文", "管理员 "+claims.Username+" 查看了资产登录凭据明文内容。")
	c.JSON(nethttp.StatusOK, gin.H{"data": data})
}

func (h *Handler) CreateAsset(c *gin.Context) {
	var input CreateAssetInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	data, err := h.service.CreateAsset(input)
	if err != nil {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(nethttp.StatusCreated, gin.H{"data": data})
}

func (h *Handler) CreateCredential(c *gin.Context) {
	var input CreateCredentialInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	data, err := h.service.CreateCredential(input)
	if err != nil {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(nethttp.StatusCreated, gin.H{"data": data})
}

func (h *Handler) CreateQuickCommand(c *gin.Context) {
	var input CreateQuickCommandInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	data, err := h.service.CreateQuickCommand(input)
	if err != nil {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(nethttp.StatusCreated, gin.H{"data": data})
}

func (h *Handler) UpdateQuickCommand(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("commandId"))
	if err != nil {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": "invalid command id"})
		return
	}
	var input UpdateQuickCommandInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	data, err := h.service.UpdateQuickCommand(uint(id), input)
	if err != nil {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(nethttp.StatusOK, gin.H{"data": data})
}

func (h *Handler) BatchCreateAssets(c *gin.Context) {
	var input BatchAssetCreateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	data, err := h.service.BatchCreateAssets(input)
	if err != nil {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(nethttp.StatusOK, gin.H{"data": data})
}

func (h *Handler) CreateGroup(c *gin.Context) {
	var input CreateAssetGroupInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	data, err := h.service.CreateGroup(input)
	if err != nil {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(nethttp.StatusCreated, gin.H{"data": data})
}

func (h *Handler) UpdateAsset(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": "invalid asset id"})
		return
	}
	var input UpdateAssetInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	data, err := h.service.UpdateAsset(uint(id), input)
	if err != nil {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(nethttp.StatusOK, gin.H{"data": data})
}

func (h *Handler) DeleteAsset(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": "invalid asset id"})
		return
	}
	if err := h.service.DeleteAsset(uint(id)); err != nil {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(nethttp.StatusOK, gin.H{"data": gin.H{"deleted": true}})
}

func (h *Handler) BatchUpdateAssets(c *gin.Context) {
	var input BatchAssetUpdateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	data, err := h.service.BatchUpdateAssets(input)
	if err != nil {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(nethttp.StatusOK, gin.H{"data": data})
}

func (h *Handler) BatchDeleteAssets(c *gin.Context) {
	var input BatchAssetDeleteInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	data, err := h.service.BatchDeleteAssets(input)
	if err != nil {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(nethttp.StatusOK, gin.H{"data": data})
}

func (h *Handler) UpdateGroup(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("groupId"))
	if err != nil {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": "invalid group id"})
		return
	}
	var input UpdateAssetGroupInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	data, err := h.service.UpdateGroup(uint(id), input)
	if err != nil {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(nethttp.StatusOK, gin.H{"data": data})
}

func (h *Handler) CreateAccount(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": "invalid asset id"})
		return
	}
	var input CreateAccountInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	data, err := h.service.CreateAccount(uint(id), input)
	if err != nil {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(nethttp.StatusCreated, gin.H{"data": data})
}

func (h *Handler) ImportCredential(c *gin.Context) {
	assetID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": "invalid asset id"})
		return
	}
	var input ImportCredentialInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	data, err := h.service.ImportCredentialToAsset(uint(assetID), input)
	if err != nil {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(nethttp.StatusCreated, gin.H{"data": data})
}

func (h *Handler) QuickConnect(c *gin.Context) {
	var input QuickConnectInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	data, err := h.service.QuickConnect(input)
	if err != nil {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(nethttp.StatusOK, gin.H{"data": data})
}

func (h *Handler) CreateTerminalTicket(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": "invalid asset id"})
		return
	}
	var input TerminalTicketInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	data, err := h.service.CreateTerminalTicket(uint(id), input)
	if err != nil {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(nethttp.StatusOK, gin.H{"data": data})
}

func (h *Handler) DeleteAccount(c *gin.Context) {
	assetID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": "invalid asset id"})
		return
	}
	accountID, err := strconv.Atoi(c.Param("accountId"))
	if err != nil {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": "invalid account id"})
		return
	}
	if err := h.service.DeleteAccount(uint(assetID), uint(accountID)); err != nil {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(nethttp.StatusOK, gin.H{"data": gin.H{"deleted": true}})
}

func (h *Handler) SetDefaultAccount(c *gin.Context) {
	assetID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": "invalid asset id"})
		return
	}
	accountID, err := strconv.Atoi(c.Param("accountId"))
	if err != nil {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": "invalid account id"})
		return
	}
	data, err := h.service.SetDefaultAccount(uint(assetID), uint(accountID))
	if err != nil {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(nethttp.StatusOK, gin.H{"data": data})
}

func (h *Handler) DeleteGroup(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("groupId"))
	if err != nil {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": "invalid group id"})
		return
	}
	if err := h.service.DeleteGroup(uint(id)); err != nil {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(nethttp.StatusOK, gin.H{"data": gin.H{"deleted": true}})
}

func (h *Handler) DeleteCredential(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("credentialId"))
	if err != nil {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": "invalid credential id"})
		return
	}
	if err := h.service.DeleteCredential(uint(id)); err != nil {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(nethttp.StatusOK, gin.H{"data": gin.H{"deleted": true}})
}

func (h *Handler) DeleteQuickCommand(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("commandId"))
	if err != nil {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": "invalid command id"})
		return
	}
	if err := h.service.DeleteQuickCommand(uint(id)); err != nil {
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(nethttp.StatusOK, gin.H{"data": gin.H{"deleted": true}})
}
