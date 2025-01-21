package services

import (
	"RoyDental/models"
	"RoyDental/repositories"
	"context"
)

type InsuranceCompanyService struct {
	repository *repositories.InsuranceCompanyRepository
}

func NewInsuranceCompanyService(repository *repositories.InsuranceCompanyRepository) *InsuranceCompanyService {
	return &InsuranceCompanyService{repository: repository}
}

func (s *InsuranceCompanyService) Create(ctx context.Context, company *models.InsuranceCompany) error {
	return s.repository.Create(ctx, company)
}

func (s *InsuranceCompanyService) GetByID(ctx context.Context, id string) (*models.InsuranceCompany, error) {
	return s.repository.GetByID(ctx, id)
}

func (s *InsuranceCompanyService) GetAll(ctx context.Context) ([]models.InsuranceCompany, error) {
	return s.repository.GetAll(ctx)
}

func (s *InsuranceCompanyService) Update(ctx context.Context, company *models.InsuranceCompany) error {
	return s.repository.Update(ctx, company)
}

func (s *InsuranceCompanyService) Delete(ctx context.Context, id string) error {
	return s.repository.Delete(ctx, id)
}
