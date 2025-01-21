package repositories

import (
	"RoyDental/cache"
	"RoyDental/database"
	"RoyDental/models"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	InsuranceCompanyCacheExpiry = 7 * 24 * time.Hour
)

type InsuranceCompanyRepository struct {
	cache *cache.Cache
}

func NewInsuranceCompanyRepository(cache *cache.Cache) *InsuranceCompanyRepository {
	return &InsuranceCompanyRepository{cache: cache}
}

func (r *InsuranceCompanyRepository) Create(ctx context.Context, company *models.InsuranceCompany) error {
	lockKey := fmt.Sprintf("insurance_company_lock:%s", company.Name)
	lockValue := uuid.New().String() // Generate a unique lock value
	// Retry logic for acquiring lock
	maxRetries := 3
	retryDelay := 2 * time.Second
	var locked bool
	var err error
	for i := 0; i < maxRetries; i++ {
		locked, err = database.NewLock(ctx, lockKey, lockValue, 10*time.Second) // Shortened expiry
		if err == nil && locked {
			break
		}
		if i < maxRetries-1 {
			time.Sleep(retryDelay)
		}
	}
	if !locked {
		return fmt.Errorf("failed to acquire lock after retries: %w", err)
	}
	defer func() {
		if err := database.ReleaseLock(ctx, lockKey, lockValue); err != nil {
			log.Printf("Failed to release lock: %v", err)
		}
	}()

	// Check if a record with the same name already exists
	var existingCompany models.InsuranceCompany
	if err := database.DB.Where("name = ?", company.Name).First(&existingCompany).Error; err == nil {
		return fmt.Errorf("insurance company with name %s already exists", company.Name)
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("failed to check for existing insurance company: %w", err)
	}

	// Obtain the next sequence value outside the transaction
	var nextID string
	if err := database.DB.Raw("SELECT 'IC-' || LPAD(nextval('insurance_company_id_seq')::TEXT, 6, '0')").Scan(&nextID).Error; err != nil {
		return fmt.Errorf("failed to obtain next sequence value: %w", err)
	}

	// Set the obtained ID to the insurance company
	company.ID = nextID

	return database.DB.Transaction(func(tx *gorm.DB) error {
		// Create the insurance company record
		if err := tx.Create(company).Error; err != nil {
			// If the creation fails, rollback the sequence
			if rollbackErr := database.DB.Exec("SELECT setval('insurance_company_id_seq', (SELECT last_value FROM insurance_company_id_seq) - 1, false)").Error; rollbackErr != nil {
				return fmt.Errorf("transaction failed and sequence rollback failed: %v, rollback error: %v", err, rollbackErr)
			}
			return fmt.Errorf("failed to create insurance company: %w", err)
		}

		// Delete cache for the newly created insurance company and all insurance companies
		if err := r.cache.Delete(ctx, r.getInsuranceCompanyCacheKey(company.ID)); err != nil {
			return fmt.Errorf("failed to delete insurance company cache: %w", err)
		}
		return r.cache.DeleteAll(ctx, "insurance_companies_cache")
	})
}

func (r *InsuranceCompanyRepository) GetByID(ctx context.Context, id string) (*models.InsuranceCompany, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	cacheKey := r.getInsuranceCompanyCacheKey(id)
	cachedCompany, err := r.cache.Get(ctx, cacheKey)
	if err == nil {
		var company models.InsuranceCompany
		if err := json.Unmarshal([]byte(cachedCompany), &company); err == nil {
			return &company, nil
		}
	} else if err != redis.Nil {
		log.Printf("Failed to get insurance company from cache: %v", err)
	}

	var company models.InsuranceCompany
	err = database.DB.Select("id, name").First(&company, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get insurance company: %w", err)
	}

	companyJSON, err := json.Marshal(company)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal insurance company: %w", err)
	}
	if err := r.cache.Set(ctx, cacheKey, companyJSON, InsuranceCompanyCacheExpiry); err != nil {
		log.Printf("Failed to set insurance company in cache: %v", err)
	}

	return &company, nil
}

func (r *InsuranceCompanyRepository) GetAll(ctx context.Context) ([]models.InsuranceCompany, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	cacheKey := "insurance_companies_cache"
	cachedCompanies, err := r.cache.Get(ctx, cacheKey)
	if err == nil {
		var companies []models.InsuranceCompany
		if err := json.Unmarshal([]byte(cachedCompanies), &companies); err == nil {
			return companies, nil
		}
	} else if err != redis.Nil {
		log.Printf("Failed to get insurance companies from cache: %v", err)
	}

	var companies []models.InsuranceCompany
	err = database.DB.
		Select("id, name").
		Order("id DESC").
		Find(&companies).
		Error
	if err != nil {
		return nil, fmt.Errorf("failed to get all insurance companies: %w", err)
	}

	companiesJSON, err := json.Marshal(companies)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal insurance companies: %w", err)
	}
	if err := r.cache.Set(ctx, cacheKey, companiesJSON, InsuranceCompanyCacheExpiry); err != nil {
		log.Printf("Failed to set insurance companies in cache: %v", err)
	}

	return companies, nil
}

func (r *InsuranceCompanyRepository) Update(ctx context.Context, company *models.InsuranceCompany) error {
	lockKey := fmt.Sprintf("insurance_company_lock:%s", company.ID)
	lockValue := uuid.New().String() // Generate a unique lock value
	// Retry logic for acquiring lock
	maxRetries := 3
	retryDelay := 2 * time.Second
	var locked bool
	var err error
	for i := 0; i < maxRetries; i++ {
		locked, err = database.NewLock(ctx, lockKey, lockValue, 10*time.Second)
		if err == nil && locked {
			break
		}
		if i < maxRetries-1 {
			time.Sleep(retryDelay)
		}
	}
	if !locked {
		return fmt.Errorf("failed to acquire lock after retries: %w", err)
	}
	defer func() {
		if err := database.ReleaseLock(ctx, lockKey, lockValue); err != nil {
			log.Printf("Failed to release lock: %v", err)
		}
	}()

	err = database.DB.Save(company).Error
	if err != nil {
		return fmt.Errorf("failed to update insurance company: %w", err)
	}
	// Delete cache for the updated insurance company and all insurance companies
	if err := r.cache.Delete(ctx, r.getInsuranceCompanyCacheKey(company.ID)); err != nil {
		return fmt.Errorf("failed to delete insurance company cache: %w", err)
	}
	return r.cache.DeleteAll(ctx, "insurance_companies_cache")
}

func (r *InsuranceCompanyRepository) Delete(ctx context.Context, id string) error {
	lockKey := fmt.Sprintf("insurance_company_lock:%s", id)
	lockValue := uuid.New().String() // Generate a unique lock value
	// Retry logic for acquiring lock
	maxRetries := 3
	retryDelay := 2 * time.Second
	var locked bool
	var err error
	for i := 0; i < maxRetries; i++ {
		locked, err = database.NewLock(ctx, lockKey, lockValue, 10*time.Second)
		if err == nil && locked {
			break
		}
		if i < maxRetries-1 {
			time.Sleep(retryDelay)
		}
	}
	if !locked {
		return fmt.Errorf("failed to acquire lock after retries: %w", err)
	}
	defer func() {
		if err := database.ReleaseLock(ctx, lockKey, lockValue); err != nil {
			log.Printf("Failed to release lock: %v", err)
		}
	}()

	err = database.DB.Delete(&models.InsuranceCompany{}, "id = ?", id).Error
	if err != nil {
		return fmt.Errorf("failed to delete insurance company: %w", err)
	}
	// Delete cache for the deleted insurance company and all insurance companies
	if err := r.cache.Delete(ctx, r.getInsuranceCompanyCacheKey(id)); err != nil {
		return fmt.Errorf("failed to delete insurance company cache: %w", err)
	}
	return r.cache.DeleteAll(ctx, "insurance_companies_cache")
}

func (r *InsuranceCompanyRepository) getInsuranceCompanyCacheKey(id string) string {
	return fmt.Sprintf("insurance_company_cache:%s", id)
}
