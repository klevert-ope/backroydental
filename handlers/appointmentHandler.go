package handlers

import (
	"RoyDental/models"
	"RoyDental/services"
	"strconv"

	"github.com/gin-gonic/gin"
)

type AppointmentHandler struct {
	service *services.AppointmentService
}

func NewAppointmentHandler(service *services.AppointmentService) *AppointmentHandler {
	return &AppointmentHandler{service: service}
}

func (h *AppointmentHandler) CreateAppointment(c *gin.Context) {
	var appointment models.Appointment
	if err := c.ShouldBindJSON(&appointment); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	if err := h.service.Create(c, &appointment); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(201, appointment)
}

func (h *AppointmentHandler) GetAppointmentByID(c *gin.Context) {
	patientID := c.Param("patient_id")
	idStr := c.Param("appointment_id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(400, gin.H{"error": "Invalid appointment ID"})
		return
	}

	appointment, err := h.service.GetByID(c, patientID, uint(id))
	if err != nil {
		c.JSON(404, gin.H{"error": "Appointment not found"})
		return
	}
	c.JSON(200, appointment)
}

func (h *AppointmentHandler) GetAllAppointments(c *gin.Context) {
	appointments, err := h.service.GetAll(c)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, appointments)
}

func (h *AppointmentHandler) UpdateAppointment(c *gin.Context) {
	patientID := c.Param("patient_id")
	idStr := c.Param("appointment_id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(400, gin.H{"error": "Invalid appointment ID"})
		return
	}

	var appointment models.Appointment
	if err := c.ShouldBindJSON(&appointment); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	appointment.PatientID = patientID
	appointment.ID = uint(id)

	if err := h.service.Update(c, &appointment); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, appointment)
}

func (h *AppointmentHandler) DeleteAppointment(c *gin.Context) {
	patientID := c.Param("patient_id")
	idStr := c.Param("appointment_id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(400, gin.H{"error": "Invalid appointment ID"})
		return
	}

	if err := h.service.Delete(c, patientID, uint(id)); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(204, gin.H{"message": "Appointment deleted"})
}
