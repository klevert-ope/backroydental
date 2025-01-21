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
	DoctorCacheExpiry = 7 * 24 * time.Hour
)

type DoctorRepository struct {
	cache *cache.Cache
}

func NewDoctorRepository(cache *cache.Cache) *DoctorRepository {
	return &DoctorRepository{cache: cache}
}

func (r *DoctorRepository) Create(ctx context.Context, doctor *models.Doctor) error {
	lockKey := fmt.Sprintf("doctor_lock:%s_%s", doctor.FirstName, doctor.LastName)
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

	// Check if a record with the same unique fields already exists
	var existingDoctor models.Doctor
	if err := database.DB.Where("first_name = ? AND last_name = ?", doctor.FirstName, doctor.LastName).First(&existingDoctor).Error; err == nil {
		return errors.New("doctor with the same name already exists")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("failed to check for existing doctor: %w", err)
	}

	// Obtain the next sequence value outside the transaction
	var nextID string
	if err := database.DB.Raw("SELECT 'DR-' || LPAD(nextval('doctor_id_seq')::TEXT, 6, '0')").Scan(&nextID).Error; err != nil {
		return fmt.Errorf("failed to obtain next sequence value: %w", err)
	}

	// Set the obtained ID to the doctor
	doctor.ID = nextID

	return database.DB.Transaction(func(tx *gorm.DB) error {
		// Create the doctor record
		if err := tx.Create(doctor).Error; err != nil {
			// If the creation fails, rollback the sequence
			if rollbackErr := database.DB.Exec("SELECT setval('doctor_id_seq', (SELECT last_value FROM doctor_id_seq) - 1, false)").Error; rollbackErr != nil {
				return fmt.Errorf("transaction failed and sequence rollback failed: %v, rollback error: %v", err, rollbackErr)
			}
			return fmt.Errorf("failed to create doctor: %w", err)
		}

		// Delete cache for the newly created doctor and all doctors
		if err := r.cache.Delete(ctx, r.getDoctorCacheKey(doctor.ID)); err != nil {
			return fmt.Errorf("failed to delete doctor cache: %w", err)
		}
		return r.cache.DeleteAll(ctx, "doctors_cache")
	})
}

func (r *DoctorRepository) GetByID(ctx context.Context, id string) (*models.Doctor, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	cacheKey := r.getDoctorCacheKey(id)
	cachedDoctor, err := r.cache.Get(ctx, cacheKey)
	if err == nil {
		var doctor models.Doctor
		if err := json.Unmarshal([]byte(cachedDoctor), &doctor); err == nil {
			return &doctor, nil
		}
	} else if err != redis.Nil {
		log.Printf("Failed to get doctor from cache: %v", err)
	}

	var doctor models.Doctor
	err = database.DB.Select("id, first_name, last_name, created_at").
		Preload("Appointments", func(db *gorm.DB) *gorm.DB {
			return db.Select("patient_id, doctor_id, date_time, created_at")
		}).
		Preload("Billings", func(db *gorm.DB) *gorm.DB {
			return db.Select("billing_id, patient_id, doctor_id, procedure, billing_amount, paid_cash_amount, paid_insurance_amount, balance, total_received, created_at")
		}).
		First(&doctor, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get doctor: %w", err)
	}

	doctorJSON, err := json.Marshal(doctor)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal doctor: %w", err)
	}
	if err := r.cache.Set(ctx, cacheKey, doctorJSON, DoctorCacheExpiry); err != nil {
		log.Printf("Failed to set doctor in cache: %v", err)
	}

	return &doctor, nil
}

func (r *DoctorRepository) GetAll(ctx context.Context) ([]models.Doctor, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	cacheKey := "doctors_cache"
	cachedDoctors, err := r.cache.Get(ctx, cacheKey)
	if err == nil {
		var doctors []models.Doctor
		if err := json.Unmarshal([]byte(cachedDoctors), &doctors); err == nil {
			return doctors, nil
		}
	} else if err != redis.Nil {
		log.Printf("Failed to get doctors from cache: %v", err)
	}

	var doctors []models.Doctor
	err = database.DB.Select("id, first_name, last_name, created_at").
		Preload("Appointments", func(db *gorm.DB) *gorm.DB {
			return db.Select("patient_id, doctor_id, date_time, created_at")
		}).
		Preload("Billings", func(db *gorm.DB) *gorm.DB {
			return db.Select("billing_id, patient_id, doctor_id, procedure, billing_amount, paid_cash_amount, paid_insurance_amount, balance, total_received, created_at")
		}).
		Order("created_at DESC").
		Find(&doctors).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get all doctors: %w", err)
	}

	doctorsJSON, err := json.Marshal(doctors)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal doctors: %w", err)
	}
	if err := r.cache.Set(ctx, cacheKey, doctorsJSON, DoctorCacheExpiry); err != nil {
		log.Printf("Failed to set doctors in cache: %v", err)
	}

	return doctors, nil
}

func (r *DoctorRepository) Update(ctx context.Context, doctor *models.Doctor) error {
	lockKey := fmt.Sprintf("doctor_lock:%s", doctor.ID)
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

	err = database.DB.Save(doctor).Error
	if err != nil {
		return fmt.Errorf("failed to update doctor: %w", err)
	}
	// Delete cache for the updated doctor and all doctors
	if err := r.cache.Delete(ctx, r.getDoctorCacheKey(doctor.ID)); err != nil {
		return fmt.Errorf("failed to delete doctor cache: %w", err)
	}
	return r.cache.DeleteAll(ctx, "doctors_cache")
}

func (r *DoctorRepository) Delete(ctx context.Context, id string) error {
	lockKey := fmt.Sprintf("doctor_lock:%s", id)
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

	err = database.DB.Delete(&models.Doctor{}, "id = ?", id).Error
	if err != nil {
		return fmt.Errorf("failed to delete doctor: %w", err)
	}
	// Delete cache for the deleted doctor and all doctors
	if err := r.cache.Delete(ctx, r.getDoctorCacheKey(id)); err != nil {
		return fmt.Errorf("failed to delete doctor cache: %w", err)
	}
	return r.cache.DeleteAll(ctx, "doctors_cache")
}

func (r *DoctorRepository) getDoctorCacheKey(id string) string {
	return fmt.Sprintf("doctor_cache:%s", id)
}
