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
	ExaminationCacheExpiry = 7 * 24 * time.Hour
)

type ExaminationRepository struct {
	cache *cache.Cache
}

func NewExaminationRepository(cache *cache.Cache) *ExaminationRepository {
	return &ExaminationRepository{cache: cache}
}

func (r *ExaminationRepository) Create(ctx context.Context, examination *models.Examination) error {
	lockKey := fmt.Sprintf("examination_lock:%d", examination.ID)
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

	err = database.DB.Create(examination).Error
	if err != nil {
		return fmt.Errorf("failed to create examination: %w", err)
	}
	// Delete cache for the newly created examination and all examinations
	if err := r.cache.Delete(ctx, r.getExaminationCacheKey(examination.PatientID, examination.ID)); err != nil {
		return fmt.Errorf("failed to delete examination cache: %w", err)
	}
	if err := r.cache.DeleteAll(ctx, "examinations_cache"); err != nil {
		return fmt.Errorf("failed to delete all examinations cache: %w", err)
	}
	// Invalidate the specific patient cache and all examinations cache
	if err := r.cache.Delete(ctx, r.getPatientCacheKey(examination.PatientID)); err != nil {
		return fmt.Errorf("failed to delete patient cache: %w", err)
	}
	return r.cache.DeleteAll(ctx, "patients_cache")
}

func (r *ExaminationRepository) GetByID(ctx context.Context, patientID string, id uint) (*models.Examination, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	cacheKey := r.getExaminationCacheKey(patientID, id)
	cachedExamination, err := r.cache.Get(ctx, cacheKey)
	if err == nil {
		var examination models.Examination
		if err := json.Unmarshal([]byte(cachedExamination), &examination); err == nil {
			return &examination, nil
		}
	} else if err != redis.Nil {
		log.Printf("Failed to get examination from cache: %v", err)
	}

	var examination models.Examination
	err = database.DB.Select("id, patient_id, report, created_at").
		Preload("Patient", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, first_name, last_name")
		}).
		First(&examination, "id = ? AND patient_id = ?", id, patientID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get examination: %w", err)
	}

	examinationJSON, err := json.Marshal(examination)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal examination: %w", err)
	}
	if err := r.cache.Set(ctx, cacheKey, examinationJSON, ExaminationCacheExpiry); err != nil {
		log.Printf("Failed to set examination in cache: %v", err)
	}

	return &examination, nil
}

func (r *ExaminationRepository) GetAll(ctx context.Context) ([]models.Examination, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	cacheKey := "examinations_cache"
	cachedExaminations, err := r.cache.Get(ctx, cacheKey)
	if err == nil {
		var examinations []models.Examination
		if err := json.Unmarshal([]byte(cachedExaminations), &examinations); err == nil {
			return examinations, nil
		}
	} else if err != redis.Nil {
		log.Printf("Failed to get examinations from cache: %v", err)
	}

	var examinations []models.Examination
	err = database.DB.Select("id, patient_id, report, created_at").
		Preload("Patient", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, first_name, last_name")
		}).
		Order("created_at DESC").
		Find(&examinations).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get all examinations: %w", err)
	}

	examinationsJSON, err := json.Marshal(examinations)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal examinations: %w", err)
	}
	if err := r.cache.Set(ctx, cacheKey, examinationsJSON, ExaminationCacheExpiry); err != nil {
		log.Printf("Failed to set examinations in cache: %v", err)
	}

	return examinations, nil
}

func (r *ExaminationRepository) Update(ctx context.Context, examination *models.Examination) error {
	lockKey := fmt.Sprintf("examination_lock:%d", examination.ID)
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

	err = database.DB.Save(examination).Error
	if err != nil {
		return fmt.Errorf("failed to update examination: %w", err)
	}
	// Delete cache for the updated examination and all examinations
	if err := r.cache.Delete(ctx, r.getExaminationCacheKey(examination.PatientID, examination.ID)); err != nil {
		return fmt.Errorf("failed to delete examination cache: %w", err)
	}
	if err := r.cache.DeleteAll(ctx, "examinations_cache"); err != nil {
		return fmt.Errorf("failed to delete all examinations cache: %w", err)
	}
	// Invalidate the specific patient cache and all examinations cache
	if err := r.cache.Delete(ctx, r.getPatientCacheKey(examination.PatientID)); err != nil {
		return fmt.Errorf("failed to delete patient cache: %w", err)
	}
	return r.cache.DeleteAll(ctx, "patients_cache")
}

func (r *ExaminationRepository) Delete(ctx context.Context, id uint) error {
	lockKey := fmt.Sprintf("examination_lock:%d", id)
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

	var examination models.Examination
	if err := database.DB.First(&examination, "id = ?", id).Error; err != nil {
		return fmt.Errorf("failed to find examination: %w", err)
	}

	err = database.DB.Delete(&models.Examination{}, "id = ?", id).Error
	if err != nil {
		return fmt.Errorf("failed to delete examination: %w", err)
	}
	// Delete cache for the deleted examination and all examinations
	if err := r.cache.Delete(ctx, r.getExaminationCacheKey(examination.PatientID, id)); err != nil {
		return fmt.Errorf("failed to delete examination cache: %w", err)
	}
	if err := r.cache.DeleteAll(ctx, "examinations_cache"); err != nil {
		return fmt.Errorf("failed to delete all examinations cache: %w", err)
	}
	// Invalidate the specific patient cache and all examinations cache
	if err := r.cache.Delete(ctx, r.getPatientCacheKey(examination.PatientID)); err != nil {
		return fmt.Errorf("failed to delete patient cache: %w", err)
	}
	return r.cache.DeleteAll(ctx, "patients_cache")
}

func (r *ExaminationRepository) DeleteCache(ctx context.Context, patientID string, id uint) error {
	return r.cache.Delete(ctx, r.getExaminationCacheKey(patientID, id))
}

func (r *ExaminationRepository) DeleteAllCache(ctx context.Context) error {
	return r.cache.DeleteAll(ctx, "examinations_cache")
}

func (r *ExaminationRepository) getExaminationCacheKey(patientID string, id uint) string {
	return fmt.Sprintf("examination_cache:%s:%d", patientID, id)
}

func (r *ExaminationRepository) getPatientCacheKey(patientID string) string {
	return fmt.Sprintf("patient_cache:%s", patientID)
}
