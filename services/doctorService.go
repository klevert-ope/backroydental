package services

import (
	"RoyDental/models"
	"RoyDental/repositories"
	"context"
)

type DoctorService struct {
	repository *repositories.DoctorRepository
}

func NewDoctorService(repository *repositories.DoctorRepository) *DoctorService {
	return &DoctorService{repository: repository}
}

func (s *DoctorService) Create(ctx context.Context, doctor *models.Doctor) error {
	return s.repository.Create(ctx, doctor)
}

func (s *DoctorService) GetByID(ctx context.Context, id string) (*models.Doctor, error) {
	return s.repository.GetByID(ctx, id)
}

func (s *DoctorService) GetAll(ctx context.Context) ([]models.Doctor, error) {
	return s.repository.GetAll(ctx)
}

func (s *DoctorService) Update(ctx context.Context, doctor *models.Doctor) error {
	return s.repository.Update(ctx, doctor)
}

func (s *DoctorService) Delete(ctx context.Context, id string) error {
	return s.repository.Delete(ctx, id)
}
