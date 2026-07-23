package blockedcustomers

import (
	"diaxel/internal/config"
	"diaxel/internal/grpc/db"
	"net/http"

	"github.com/gin-gonic/gin"
)

type BlockedCustomersHandler struct {
	cfg *config.Settings
	db  *db.Client
}

func NewBlockedCustomersHandler(cfg *config.Settings, db *db.Client) *BlockedCustomersHandler {
	return &BlockedCustomersHandler{cfg: cfg, db: db}
}

type BlockCustomerRequest struct {
	UserID string `json:"user_id" binding:"required"`
}

func (h *BlockedCustomersHandler) BlockCustomer(c *gin.Context) {
	var req BlockCustomerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	err := h.db.BlockCustomer(req.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to block customer"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *BlockedCustomersHandler) GetAllBlockedCustomers(c *gin.Context) {
	userIDs, err := h.db.GetAllBlockedCustomers()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get blocked customers"})
		return
	}

	var answer []BlockCustomerRequest
	for _, id := range userIDs {
		answer = append(answer, BlockCustomerRequest{UserID: id})
	}

	// Returning empty array if nil to match typical API structures
	if answer == nil {
		answer = []BlockCustomerRequest{}
	}

	c.JSON(http.StatusOK, gin.H{"answer": answer})
}
