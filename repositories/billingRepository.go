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
	BillingCacheExpiry = 7 * 24 * time.Hour
)

type BillingRepository struct {
	cache *cache.Cache
}

func NewBillingRepository(cache *cache.Cache) *BillingRepository {
	return &BillingRepository{cache: cache}
}

func (r *BillingRepository) Create(ctx context.Context, billing *models.Billing) error {
	lockKey := fmt.Sprintf("billing_lock:%s", billing.BillingID)
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

	// Check if the doctor exists
	var doctor models.Doctor
	if err := database.DB.First(&doctor, "id = ?", billing.DoctorID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("doctor not found")
		}
		return fmt.Errorf("failed to find doctor: %w", err)
	}

	// Obtain the next sequence value outside the transaction
	var nextID string
	if err := database.DB.Raw("SELECT 'PB-' || LPAD(nextval('billing_id_seq')::TEXT, 6, '0')").Scan(&nextID).Error; err != nil {
		return fmt.Errorf("failed to obtain next sequence value: %w", err)
	}

	// Set the obtained ID to the billing
	billing.BillingID = nextID

	// Calculate the balance and total_received
	billing.Balance = billing.BillingAmount - (billing.PaidCashAmount + billing.PaidInsuranceAmount)
	billing.TotalReceived = billing.PaidCashAmount + billing.PaidInsuranceAmount

	return database.DB.Transaction(func(tx *gorm.DB) error {
		// Create the billing record
		if err := tx.Create(billing).Error; err != nil {
			// If the creation fails, rollback the sequence
			if rollbackErr := database.DB.Exec("SELECT setval('billing_id_seq', (SELECT last_value FROM billing_id_seq) - 1, false)").Error; rollbackErr != nil {
				return fmt.Errorf("transaction failed and sequence rollback failed: %v, rollback error: %v", err, rollbackErr)
			}
			return fmt.Errorf("failed to create billing: %w", err)
		}

		// Delete cache for the newly created billing and all billings
		if err := r.cache.Delete(ctx, r.getBillingCacheKey(billing.BillingID)); err != nil {
			return fmt.Errorf("failed to delete billing cache: %w", err)
		}
		if err := r.cache.DeleteAll(ctx, "billings_cache"); err != nil {
			return fmt.Errorf("failed to delete all billings cache: %w", err)
		}
		// Invalidate the specific patient cache and all billings cache
		if err := r.cache.Delete(ctx, r.getPatientCacheKey(billing.PatientID)); err != nil {
			return fmt.Errorf("failed to delete patient cache: %w", err)
		}
		return r.cache.DeleteAll(ctx, "patients_cache")
	})
}

func (r *BillingRepository) GetByID(ctx context.Context, id string) (*models.Billing, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	cacheKey := r.getBillingCacheKey(id)
	cachedBilling, err := r.cache.Get(ctx, cacheKey)
	if err == nil {
		var billing models.Billing
		if err := json.Unmarshal([]byte(cachedBilling), &billing); err == nil {
			return &billing, nil
		}
	} else if err != redis.Nil {
		log.Printf("Failed to get billing from cache: %v", err)
	}

	var billing models.Billing
	err = database.DB.Select("billing_id, patient_id, doctor_id, procedure, billing_amount, paid_cash_amount, paid_insurance_amount, balance, total_received, created_at").
		Preload("Patient", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, first_name, last_name")
		}).
		Preload("Doctor", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, first_name, last_name")
		}).
		First(&billing, "billing_id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get billing: %w", err)
	}

	billingJSON, err := json.Marshal(billing)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal billing: %w", err)
	}
	if err := r.cache.Set(ctx, cacheKey, billingJSON, BillingCacheExpiry); err != nil {
		log.Printf("Failed to set billing in cache: %v", err)
	}

	return &billing, nil
}

func (r *BillingRepository) GetAll(ctx context.Context) ([]models.Billing, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	cacheKey := "billings_cache"
	cachedBillings, err := r.cache.Get(ctx, cacheKey)
	if err == nil {
		var billings []models.Billing
		if err := json.Unmarshal([]byte(cachedBillings), &billings); err == nil {
			return billings, nil
		}
	} else if err != redis.Nil {
		log.Printf("Failed to get billings from cache: %v", err)
	}

	var billings []models.Billing
	err = database.DB.Select("billing_id, patient_id, doctor_id, procedure, billing_amount, paid_cash_amount, paid_insurance_amount, balance, total_received, created_at").
		Preload("Patient", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, first_name, last_name")
		}).
		Preload("Doctor", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, first_name, last_name")
		}).
		Order("created_at DESC").
		Find(&billings).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get all billings: %w", err)
	}

	billingsJSON, err := json.Marshal(billings)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal billings: %w", err)
	}
	if err := r.cache.Set(ctx, cacheKey, billingsJSON, BillingCacheExpiry); err != nil {
		log.Printf("Failed to set billings in cache: %v", err)
	}

	return billings, nil
}

func (r *BillingRepository) Update(ctx context.Context, billing *models.Billing) error {
	lockKey := fmt.Sprintf("billing_lock:%s", billing.BillingID)
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

	// Check if the doctor exists
	var doctor models.Doctor
	if err := database.DB.First(&doctor, "id = ?", billing.DoctorID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("doctor not found")
		}
		return fmt.Errorf("failed to find doctor: %w", err)
	}

	// Calculate the balance and total_received
	billing.Balance = billing.BillingAmount - (billing.PaidCashAmount + billing.PaidInsuranceAmount)
	billing.TotalReceived = billing.PaidCashAmount + billing.PaidInsuranceAmount

	err = database.DB.Save(billing).Error
	if err != nil {
		return fmt.Errorf("failed to update billing: %w", err)
	}
	// Delete cache for the updated billing and all billings
	if err := r.cache.Delete(ctx, r.getBillingCacheKey(billing.BillingID)); err != nil {
		return fmt.Errorf("failed to delete billing cache: %w", err)
	}
	if err := r.cache.DeleteAll(ctx, "billings_cache"); err != nil {
		return fmt.Errorf("failed to delete all billings cache: %w", err)
	}
	// Invalidate the specific patient cache and all billings cache
	if err := r.cache.Delete(ctx, r.getPatientCacheKey(billing.PatientID)); err != nil {
		return fmt.Errorf("failed to delete patient cache: %w", err)
	}
	return r.cache.DeleteAll(ctx, "patients_cache")
}

func (r *BillingRepository) Delete(ctx context.Context, id string) error {
	lockKey := fmt.Sprintf("billing_lock:%s", id)
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

	var billing models.Billing
	if err := database.DB.First(&billing, "billing_id = ?", id).Error; err != nil {
		return fmt.Errorf("failed to find billing: %w", err)
	}

	err = database.DB.Delete(&models.Billing{}, "billing_id = ?", id).Error
	if err != nil {
		return fmt.Errorf("failed to delete billing: %w", err)
	}
	// Delete cache for the deleted billing and all billings
	if err := r.cache.Delete(ctx, r.getBillingCacheKey(id)); err != nil {
		return fmt.Errorf("failed to delete billing cache: %w", err)
	}
	if err := r.cache.DeleteAll(ctx, "billings_cache"); err != nil {
		return fmt.Errorf("failed to delete all billings cache: %w", err)
	}
	// Invalidate the specific patient cache and all billings cache
	if err := r.cache.Delete(ctx, r.getPatientCacheKey(billing.PatientID)); err != nil {
		return fmt.Errorf("failed to delete patient cache: %w", err)
	}
	return r.cache.DeleteAll(ctx, "patients_cache")
}

func (r *BillingRepository) DeleteCache(ctx context.Context, id string) error {
	return r.cache.Delete(ctx, r.getBillingCacheKey(id))
}

func (r *BillingRepository) DeleteAllCache(ctx context.Context) error {
	return r.cache.DeleteAll(ctx, "billings_cache")
}

func (r *BillingRepository) getBillingCacheKey(id string) string {
	return fmt.Sprintf("billing_cache:%s", id)
}

func (r *BillingRepository) getPatientCacheKey(patientID string) string {
	return fmt.Sprintf("patient_cache:%s", patientID)
}
