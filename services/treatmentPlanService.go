package services

import (
	"RoyDental/models"
	"RoyDental/repositories"
	"context"
)

type TreatmentPlanService struct {
	repository *repositories.TreatmentPlanRepository
}

func NewTreatmentPlanService(repository *repositories.TreatmentPlanRepository) *TreatmentPlanService {
	return &TreatmentPlanService{repository: repository}
}

func (s *TreatmentPlanService) Create(ctx context.Context, plan *models.TreatmentPlan) error {
	return s.repository.Create(ctx, plan)
}

func (s *TreatmentPlanService) GetByID(ctx context.Context, patientID string, id uint) (*models.TreatmentPlan, error) {
	return s.repository.GetByID(ctx, patientID, id)
}

func (s *TreatmentPlanService) GetAll(ctx context.Context) ([]models.TreatmentPlan, error) {
	return s.repository.GetAll(ctx)
}

func (s *TreatmentPlanService) Update(ctx context.Context, plan *models.TreatmentPlan) error {
	return s.repository.Update(ctx, plan)
}

func (s *TreatmentPlanService) Delete(ctx context.Context, patientID string, id uint) error {
	return s.repository.Delete(ctx, patientID, id)
}
