package services

import (
	"RoyDental/models"
	"RoyDental/repositories"
	"context"
)

type EmergencyContactService struct {
	repository *repositories.EmergencyContactRepository
}

func NewEmergencyContactService(repository *repositories.EmergencyContactRepository) *EmergencyContactService {
	return &EmergencyContactService{repository: repository}
}

func (s *EmergencyContactService) Create(ctx context.Context, contact *models.EmergencyContact) error {
	return s.repository.Create(ctx, contact)
}

func (s *EmergencyContactService) GetByID(ctx context.Context, patientID string, id uint) (*models.EmergencyContact, error) {
	return s.repository.GetByID(ctx, patientID, id)
}

func (s *EmergencyContactService) GetAll(ctx context.Context) ([]models.EmergencyContact, error) {
	return s.repository.GetAll(ctx)
}

func (s *EmergencyContactService) Update(ctx context.Context, contact *models.EmergencyContact) error {
	return s.repository.Update(ctx, contact)
}

func (s *EmergencyContactService) Delete(ctx context.Context, patientID string, id uint) error {
	return s.repository.Delete(ctx, patientID, id)
}
