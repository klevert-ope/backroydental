package handlers

import (
	"RoyDental/models"
	"RoyDental/services"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type TreatmentPlanHandler struct {
	service *services.TreatmentPlanService
}

func NewTreatmentPlanHandler(service *services.TreatmentPlanService) *TreatmentPlanHandler {
	return &TreatmentPlanHandler{service: service}
}

func (h *TreatmentPlanHandler) CreateTreatmentPlan(c *gin.Context) {
	var plan models.TreatmentPlan
	if err := c.ShouldBindJSON(&plan); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.service.Create(c, &plan); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, plan)
}

func (h *TreatmentPlanHandler) GetTreatmentPlanByID(c *gin.Context) {
	patientID := c.Param("patient_id")
	id, err := strconv.ParseUint(c.Param("treatment_plan_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}
	plan, err := h.service.GetByID(c, patientID, uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Treatment Plan not found"})
		return
	}
	c.JSON(http.StatusOK, plan)
}

func (h *TreatmentPlanHandler) GetAllTreatmentPlans(c *gin.Context) {
	plans, err := h.service.GetAll(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, plans)
}

func (h *TreatmentPlanHandler) UpdateTreatmentPlan(c *gin.Context) {
	patientID := c.Param("patient_id")
	id, err := strconv.ParseUint(c.Param("treatment_plan_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}
	var plan models.TreatmentPlan
	if err := c.ShouldBindJSON(&plan); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	plan.ID = uint(id)
	plan.PatientID = patientID
	if err := h.service.Update(c, &plan); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, plan)
}

func (h *TreatmentPlanHandler) DeleteTreatmentPlan(c *gin.Context) {
	patientID := c.Param("patient_id")
	id, err := strconv.ParseUint(c.Param("treatment_plan_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}
	if err := h.service.Delete(c, patientID, uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusNoContent, gin.H{"message": "Treatment Plan deleted"})
}
