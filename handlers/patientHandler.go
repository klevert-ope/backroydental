package handlers

import (
	"RoyDental/models"
	"RoyDental/services"

	"github.com/gin-gonic/gin"
)

type PatientHandler struct {
	service *services.PatientService
}

func NewPatientHandler(service *services.PatientService) *PatientHandler {
	return &PatientHandler{service: service}
}

func (h *PatientHandler) CreatePatient(c *gin.Context) {
	var patient models.Patient
	if err := c.ShouldBindJSON(&patient); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	if err := h.service.Create(c, &patient); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(201, patient)
}

func (h *PatientHandler) GetPatientByID(c *gin.Context) {
	id := c.Param("patient_id")
	patient, err := h.service.GetByID(c, id)
	if err != nil {
		c.JSON(404, gin.H{"error": "Patient not found"})
		return
	}
	c.JSON(200, patient)
}

func (h *PatientHandler) GetAllPatients(c *gin.Context) {
	patients, err := h.service.GetAll(c)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, patients)
}

func (h *PatientHandler) UpdatePatient(c *gin.Context) {
	id := c.Param("patient_id")
	var patient models.Patient
	if err := c.ShouldBindJSON(&patient); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	patient.ID = id
	if err := h.service.Update(c, &patient); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, patient)
}

func (h *PatientHandler) DeletePatient(c *gin.Context) {
	id := c.Param("patient_id")
	if err := h.service.Delete(c, id); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(204, gin.H{"message": "Patient deleted"})
}

func (h *PatientHandler) DeletePatientAndRelated(c *gin.Context) {
	id := c.Param("patient_id")
	if err := h.service.DeletePatientAndRelated(c, id); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(204, gin.H{"message": "Patient and all related records deleted"})
}
