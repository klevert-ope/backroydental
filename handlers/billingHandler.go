package handlers

import (
	"RoyDental/models"
	"RoyDental/services"

	"github.com/gin-gonic/gin"
)

type BillingHandler struct {
	service *services.BillingService
}

func NewBillingHandler(service *services.BillingService) *BillingHandler {
	return &BillingHandler{service: service}
}

func (h *BillingHandler) CreateBilling(c *gin.Context) {
	var billing models.Billing
	if err := c.ShouldBindJSON(&billing); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	if err := h.service.Create(c, &billing); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(201, billing)
}

func (h *BillingHandler) GetBillingByID(c *gin.Context) {
	id := c.Param("id")
	billing, err := h.service.GetByID(c, id)
	if err != nil {
		c.JSON(404, gin.H{"error": "Billing not found"})
		return
	}
	c.JSON(200, billing)
}

func (h *BillingHandler) GetAllBillings(c *gin.Context) {
	billings, err := h.service.GetAll(c)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, billings)
}

func (h *BillingHandler) UpdateBilling(c *gin.Context) {
	id := c.Param("id")
	var billing models.Billing
	if err := c.ShouldBindJSON(&billing); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	billing.BillingID = id
	if err := h.service.Update(c, &billing); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, billing)
}

func (h *BillingHandler) DeleteBilling(c *gin.Context) {
	id := c.Param("id")
	if err := h.service.Delete(c, id); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(204, gin.H{"message": "Billing deleted"})
}
