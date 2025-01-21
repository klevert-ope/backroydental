package handlers

import (
	"RoyDental/models"
	"RoyDental/services"

	"github.com/gin-gonic/gin"
)

type DoctorHandler struct {
	service *services.DoctorService
}

func NewDoctorHandler(service *services.DoctorService) *DoctorHandler {
	return &DoctorHandler{service: service}
}

func (h *DoctorHandler) CreateDoctor(c *gin.Context) {
	var doctor models.Doctor
	if err := c.ShouldBindJSON(&doctor); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	if err := h.service.Create(c, &doctor); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(201, doctor)
}

func (h *DoctorHandler) GetDoctorByID(c *gin.Context) {
	id := c.Param("id")
	doctor, err := h.service.GetByID(c, id)
	if err != nil {
		c.JSON(404, gin.H{"error": "Doctor not found"})
		return
	}
	c.JSON(200, doctor)
}

func (h *DoctorHandler) GetAllDoctors(c *gin.Context) {
	doctors, err := h.service.GetAll(c)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, doctors)
}

func (h *DoctorHandler) UpdateDoctor(c *gin.Context) {
	id := c.Param("id")
	var doctor models.Doctor
	if err := c.ShouldBindJSON(&doctor); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	doctor.ID = id
	if err := h.service.Update(c, &doctor); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, doctor)
}

func (h *DoctorHandler) DeleteDoctor(c *gin.Context) {
	id := c.Param("id")
	if err := h.service.Delete(c, id); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(204, gin.H{"message": "Doctor deleted"})
}
