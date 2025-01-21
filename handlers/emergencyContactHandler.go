package handlers

import (
	"RoyDental/models"
	"RoyDental/services"
	"strconv"

	"github.com/gin-gonic/gin"
)

type EmergencyContactHandler struct {
	service *services.EmergencyContactService
}

// NewEmergencyContactHandler initializes a new EmergencyContactHandler.
func NewEmergencyContactHandler(service *services.EmergencyContactService) *EmergencyContactHandler {
	return &EmergencyContactHandler{service: service}
}

// CreateEmergencyContact handles creating a new emergency contact.
func (h *EmergencyContactHandler) CreateEmergencyContact(c *gin.Context) {
	var contact models.EmergencyContact
	if err := c.ShouldBindJSON(&contact); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	if err := h.service.Create(c, &contact); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(201, contact)
}

// GetEmergencyContactByID retrieves an emergency contact by ID.
func (h *EmergencyContactHandler) GetEmergencyContactByID(c *gin.Context) {
	patientID := c.Param("patient_id")
	idParam := c.Param("emergency_contact_id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(400, gin.H{"error": "Invalid ID"})
		return
	}
	contact, err := h.service.GetByID(c, patientID, uint(id))
	if err != nil {
		c.JSON(404, gin.H{"error": "Emergency contact not found"})
		return
	}
	c.JSON(200, contact)
}

// GetAllEmergencyContacts retrieves all emergency contacts.
func (h *EmergencyContactHandler) GetAllEmergencyContacts(c *gin.Context) {
	contacts, err := h.service.GetAll(c)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, contacts)
}

// UpdateEmergencyContact updates an existing emergency contact.
func (h *EmergencyContactHandler) UpdateEmergencyContact(c *gin.Context) {
	patientID := c.Param("patient_id")
	idParam := c.Param("emergency_contact_id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(400, gin.H{"error": "Invalid ID"})
		return
	}
	var contact models.EmergencyContact
	if err := c.ShouldBindJSON(&contact); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	contact.ID = uint(id)
	contact.PatientID = patientID
	if err := h.service.Update(c, &contact); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, contact)
}

// DeleteEmergencyContact deletes an existing emergency contact.
func (h *EmergencyContactHandler) DeleteEmergencyContact(c *gin.Context) {
	patientID := c.Param("patient_id") // Extract patient_id
	idParam := c.Param("emergency_contact_id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(400, gin.H{"error": "Invalid ID"})
		return
	}
	// Convert *gin.Context to context.Context if necessary
	if err := h.service.Delete(c.Request.Context(), patientID, uint(id)); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(204, gin.H{"message": "Emergency contact deleted"})
}
