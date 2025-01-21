package handlers

import (
	"RoyDental/models"
	"RoyDental/services"

	"github.com/gin-gonic/gin"
)

type InsuranceCompanyHandler struct {
	service *services.InsuranceCompanyService
}

func NewInsuranceCompanyHandler(service *services.InsuranceCompanyService) *InsuranceCompanyHandler {
	return &InsuranceCompanyHandler{service: service}
}

func (h *InsuranceCompanyHandler) CreateInsuranceCompany(c *gin.Context) {
	var company models.InsuranceCompany
	if err := c.ShouldBindJSON(&company); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	if err := h.service.Create(c, &company); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(201, company)
}

func (h *InsuranceCompanyHandler) GetInsuranceCompanyByID(c *gin.Context) {
	id := c.Param("id")
	company, err := h.service.GetByID(c, id)
	if err != nil {
		c.JSON(404, gin.H{"error": "Insurance Company not found"})
		return
	}
	c.JSON(200, company)
}

func (h *InsuranceCompanyHandler) GetAllInsuranceCompanies(c *gin.Context) {
	companies, err := h.service.GetAll(c)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, companies)
}

func (h *InsuranceCompanyHandler) UpdateInsuranceCompany(c *gin.Context) {
	id := c.Param("id")
	var company models.InsuranceCompany
	if err := c.ShouldBindJSON(&company); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	company.ID = id
	if err := h.service.Update(c, &company); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, company)
}

func (h *InsuranceCompanyHandler) DeleteInsuranceCompany(c *gin.Context) {
	id := c.Param("id")
	if err := h.service.Delete(c, id); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(204, gin.H{"message": "Insurance Company deleted"})
}
