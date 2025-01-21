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
	TreatmentPlanCacheExpiry = 7 * 24 * time.Hour
)

type TreatmentPlanRepository struct {
	cache *cache.Cache
}

func NewTreatmentPlanRepository(cache *cache.Cache) *TreatmentPlanRepository {
	return &TreatmentPlanRepository{cache: cache}
}

func (r *TreatmentPlanRepository) Create(ctx context.Context, plan *models.TreatmentPlan) error {
	lockKey := fmt.Sprintf("treatment_plan_lock:%s", plan.PatientID)
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

	err = database.DB.Create(plan).Error
	if err != nil {
		return fmt.Errorf("failed to create treatment plan: %w", err)
	}
	// Delete cache for the newly created treatment plan and all treatment plans
	if err := r.cache.Delete(ctx, r.getTreatmentPlanCacheKey(plan.PatientID, plan.ID)); err != nil {
		return fmt.Errorf("failed to delete treatment plan cache: %w", err)
	}
	if err := r.cache.DeleteAll(ctx, "treatment_plans_cache"); err != nil {
		return fmt.Errorf("failed to delete all treatment plans cache: %w", err)
	}
	// Invalidate the specific patient cache and all treatment plans cache
	if err := r.cache.Delete(ctx, r.getPatientCacheKey(plan.PatientID)); err != nil {
		return fmt.Errorf("failed to delete patient cache: %w", err)
	}
	return r.cache.DeleteAll(ctx, "patients_cache")
}

func (r *TreatmentPlanRepository) GetByID(ctx context.Context, patientID string, id uint) (*models.TreatmentPlan, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	cacheKey := r.getTreatmentPlanCacheKey(patientID, id)
	cachedPlan, err := r.cache.Get(ctx, cacheKey)
	if err == nil {
		var plan models.TreatmentPlan
		if err := json.Unmarshal([]byte(cachedPlan), &plan); err == nil {
			return &plan, nil
		}
	} else if err != redis.Nil {
		log.Printf("Failed to get treatment plan from cache: %v", err)
	}

	var plan models.TreatmentPlan
	err = database.DB.Select("id, patient_id, plan, created_at").
		Preload("Patient", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, first_name, last_name")
		}).
		First(&plan, "patient_id = ? AND id = ?", patientID, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get treatment plan: %w", err)
	}

	planJSON, err := json.Marshal(plan)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal treatment plan: %w", err)
	}
	if err := r.cache.Set(ctx, cacheKey, planJSON, TreatmentPlanCacheExpiry); err != nil {
		log.Printf("Failed to set treatment plan in cache: %v", err)
	}

	return &plan, nil
}

func (r *TreatmentPlanRepository) GetAll(ctx context.Context) ([]models.TreatmentPlan, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	cacheKey := "treatment_plans_cache"
	cachedPlans, err := r.cache.Get(ctx, cacheKey)
	if err == nil {
		var plans []models.TreatmentPlan
		if err := json.Unmarshal([]byte(cachedPlans), &plans); err == nil {
			return plans, nil
		}
	} else if err != redis.Nil {
		log.Printf("Failed to get treatment plans from cache: %v", err)
	}

	var plans []models.TreatmentPlan
	err = database.DB.Select("id, patient_id, plan, created_at").
		Preload("Patient", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, first_name, last_name")
		}).
		Order("created_at DESC").
		Find(&plans).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get all treatment plans: %w", err)
	}

	plansJSON, err := json.Marshal(plans)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal treatment plans: %w", err)
	}
	if err := r.cache.Set(ctx, cacheKey, plansJSON, TreatmentPlanCacheExpiry); err != nil {
		log.Printf("Failed to set treatment plans in cache: %v", err)
	}

	return plans, nil
}

func (r *TreatmentPlanRepository) Update(ctx context.Context, plan *models.TreatmentPlan) error {
	lockKey := fmt.Sprintf("treatment_plan_lock:%s", plan.PatientID)
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

	err = database.DB.Save(plan).Error
	if err != nil {
		return fmt.Errorf("failed to update treatment plan: %w", err)
	}
	// Delete cache for the updated treatment plan and all treatment plans
	if err := r.cache.Delete(ctx, r.getTreatmentPlanCacheKey(plan.PatientID, plan.ID)); err != nil {
		return fmt.Errorf("failed to delete treatment plan cache: %w", err)
	}
	if err := r.cache.DeleteAll(ctx, "treatment_plans_cache"); err != nil {
		return fmt.Errorf("failed to delete all treatment plans cache: %w", err)
	}
	// Invalidate the specific patient cache and all treatment plans cache
	if err := r.cache.Delete(ctx, r.getPatientCacheKey(plan.PatientID)); err != nil {
		return fmt.Errorf("failed to delete patient cache: %w", err)
	}
	return r.cache.DeleteAll(ctx, "patients_cache")
}

func (r *TreatmentPlanRepository) Delete(ctx context.Context, patientID string, id uint) error {
	lockKey := fmt.Sprintf("treatment_plan_lock:%s", patientID)
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

	err = database.DB.Delete(&models.TreatmentPlan{}, "patient_id = ? AND id = ?", patientID, id).Error
	if err != nil {
		return fmt.Errorf("failed to delete treatment plan: %w", err)
	}
	// Delete cache for the deleted treatment plan and all treatment plans
	if err := r.cache.Delete(ctx, r.getTreatmentPlanCacheKey(patientID, id)); err != nil {
		return fmt.Errorf("failed to delete treatment plan cache: %w", err)
	}
	if err := r.cache.DeleteAll(ctx, "treatment_plans_cache"); err != nil {
		return fmt.Errorf("failed to delete all treatment plans cache: %w", err)
	}
	// Invalidate the specific patient cache and all treatment plans cache
	if err := r.cache.Delete(ctx, r.getPatientCacheKey(patientID)); err != nil {
		return fmt.Errorf("failed to delete patient cache: %w", err)
	}
	return r.cache.DeleteAll(ctx, "patients_cache")
}

func (r *TreatmentPlanRepository) DeleteCache(ctx context.Context, patientID string, id uint) error {
	return r.cache.Delete(ctx, r.getTreatmentPlanCacheKey(patientID, id))
}

func (r *TreatmentPlanRepository) DeleteAllCache(ctx context.Context) error {
	return r.cache.DeleteAll(ctx, "treatment_plans_cache")
}

func (r *TreatmentPlanRepository) getTreatmentPlanCacheKey(patientID string, id uint) string {
	return fmt.Sprintf("treatment_plan_cache:%s:%d", patientID, id)
}

func (r *TreatmentPlanRepository) getPatientCacheKey(patientID string) string {
	return fmt.Sprintf("patient_cache:%s", patientID)
}
