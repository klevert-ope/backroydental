package services

import (
	"RoyDental/models"
	"RoyDental/repositories"
	"context"
)

type ExaminationService struct {
	repository *repositories.ExaminationRepository
}

func NewExaminationService(repository *repositories.ExaminationRepository) *ExaminationService {
	return &ExaminationService{repository: repository}
}

func (s *ExaminationService) Create(ctx context.Context, examination *models.Examination) error {
	return s.repository.Create(ctx, examination)
}

func (s *ExaminationService) GetByID(ctx context.Context, patientID string, id uint) (*models.Examination, error) {
	return s.repository.GetByID(ctx, patientID, id)
}

func (s *ExaminationService) GetAll(ctx context.Context) ([]models.Examination, error) {
	return s.repository.GetAll(ctx)
}

func (s *ExaminationService) Update(ctx context.Context, examination *models.Examination) error {
	return s.repository.Update(ctx, examination)
}

func (s *ExaminationService) Delete(ctx context.Context, id uint) error {
	return s.repository.Delete(ctx, id)
}
