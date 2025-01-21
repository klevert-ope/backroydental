package services

import (
	"RoyDental/models"
	"RoyDental/repositories"
	"context"
)

type BillingService struct {
	repository *repositories.BillingRepository
}

func NewBillingService(repository *repositories.BillingRepository) *BillingService {
	return &BillingService{repository: repository}
}

func (s *BillingService) Create(ctx context.Context, billing *models.Billing) error {
	return s.repository.Create(ctx, billing)
}

func (s *BillingService) GetByID(ctx context.Context, id string) (*models.Billing, error) {
	return s.repository.GetByID(ctx, id)
}

func (s *BillingService) GetAll(ctx context.Context) ([]models.Billing, error) {
	return s.repository.GetAll(ctx)
}

func (s *BillingService) Update(ctx context.Context, billing *models.Billing) error {
	return s.repository.Update(ctx, billing)
}

func (s *BillingService) Delete(ctx context.Context, id string) error {
	return s.repository.Delete(ctx, id)
}
