package handlers

import (
	"RoyDental/models"
	"RoyDental/services"
	"strconv"

	"github.com/gin-gonic/gin"
)

type ExaminationHandler struct {
	service *services.ExaminationService
}

func NewExaminationHandler(service *services.ExaminationService) *ExaminationHandler {
	return &ExaminationHandler{service: service}
}

func (h *ExaminationHandler) CreateExamination(c *gin.Context) {
	var examination models.Examination
	if err := c.ShouldBindJSON(&examination); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	if err := h.service.Create(c, &examination); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(201, examination)
}

func (h *ExaminationHandler) GetExaminationByID(c *gin.Context) {
	patientID := c.Param("patient_id")
	idParam := c.Param("examination_id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(400, gin.H{"error": "Invalid ID"})
		return
	}
	examination, err := h.service.GetByID(c, patientID, uint(id))
	if err != nil {
		c.JSON(404, gin.H{"error": "Examination not found"})
		return
	}
	c.JSON(200, examination)
}

func (h *ExaminationHandler) GetAllExaminations(c *gin.Context) {
	examinations, err := h.service.GetAll(c)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, examinations)
}

func (h *ExaminationHandler) UpdateExamination(c *gin.Context) {
	patientID := c.Param("patient_id")
	idParam := c.Param("examination_id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(400, gin.H{"error": "Invalid ID"})
		return
	}
	var examination models.Examination
	if err := c.ShouldBindJSON(&examination); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	examination.ID = uint(id)
	examination.PatientID = patientID
	if err := h.service.Update(c, &examination); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, examination)
}

func (h *ExaminationHandler) DeleteExamination(c *gin.Context) {
	idParam := c.Param("examination_id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(400, gin.H{"error": "Invalid ID"})
		return
	}
	if err := h.service.Delete(c, uint(id)); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(204, gin.H{"message": "Examination deleted"})
}
