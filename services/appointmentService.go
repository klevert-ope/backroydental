package services

import (
	"RoyDental/models"
	"RoyDental/repositories"
	"context"
)

type AppointmentService struct {
	repository *repositories.AppointmentRepository
}

func NewAppointmentService(repository *repositories.AppointmentRepository) *AppointmentService {
	return &AppointmentService{repository: repository}
}

func (s *AppointmentService) Create(ctx context.Context, appointment *models.Appointment) error {
	return s.repository.Create(ctx, appointment)
}

func (s *AppointmentService) GetByID(ctx context.Context, patientID string, id uint) (*models.Appointment, error) {
	return s.repository.GetByID(ctx, patientID, id)
}

func (s *AppointmentService) GetAll(ctx context.Context) ([]models.Appointment, error) {
	return s.repository.GetAll(ctx)
}

func (s *AppointmentService) Update(ctx context.Context, appointment *models.Appointment) error {
	return s.repository.Update(ctx, appointment)
}

func (s *AppointmentService) Delete(ctx context.Context, patientID string, id uint) error {
	return s.repository.Delete(ctx, patientID, id)
}
