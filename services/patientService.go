package services

import (
	"RoyDental/models"
	"RoyDental/repositories"
	"context"
)

type PatientService struct {
	repository *repositories.PatientRepository
}

func NewPatientService(repository *repositories.PatientRepository) *PatientService {
	return &PatientService{repository: repository}
}

func (s *PatientService) Create(ctx context.Context, patient *models.Patient) error {
	return s.repository.Create(ctx, patient)
}

func (s *PatientService) GetByID(ctx context.Context, id string) (*models.Patient, error) {
	return s.repository.GetByID(ctx, id)
}

func (s *PatientService) GetAll(ctx context.Context) ([]models.Patient, error) {
	return s.repository.GetAll(ctx)
}

func (s *PatientService) Update(ctx context.Context, patient *models.Patient) error {
	return s.repository.Update(ctx, patient)
}

func (s *PatientService) Delete(ctx context.Context, id string) error {
	return s.repository.Delete(ctx, id)
}

func (s *PatientService) DeletePatientAndRelated(ctx context.Context, id string) error {
	return s.repository.DeletePatientAndRelated(ctx, id)
}
