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
	AppointmentCacheExpiry = 7 * 24 * time.Hour
)

type AppointmentRepository struct {
	cache *cache.Cache
}

func NewAppointmentRepository(cache *cache.Cache) *AppointmentRepository {
	return &AppointmentRepository{cache: cache}
}

func (r *AppointmentRepository) Create(ctx context.Context, appointment *models.Appointment) error {
	lockKey := fmt.Sprintf("appointment_lock:%s_%d", appointment.PatientID, appointment.ID)
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

	// Validate the Status field
	if appointment.Status != "scheduled" && appointment.Status != "fulfilled" && appointment.Status != "cancelled" {
		return errors.New("invalid status value")
	}

	err = database.DB.Create(appointment).Error
	if err != nil {
		return fmt.Errorf("failed to create appointment: %w", err)
	}
	if err := r.cache.Delete(ctx, r.getAppointmentCacheKey(appointment.PatientID, appointment.ID)); err != nil {
		return fmt.Errorf("failed to delete appointment cache: %w", err)
	}
	if err := r.cache.DeleteAll(ctx, "appointments_cache"); err != nil {
		return fmt.Errorf("failed to delete all appointments cache: %w", err)
	}
	// Invalidate the specific patient cache and all appointments cache
	if err := r.cache.Delete(ctx, r.getPatientCacheKey(appointment.PatientID)); err != nil {
		return fmt.Errorf("failed to delete patient cache: %w", err)
	}
	return r.cache.DeleteAll(ctx, "patients_cache")
}

func (r *AppointmentRepository) GetByID(ctx context.Context, patientID string, id uint) (*models.Appointment, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	cacheKey := r.getAppointmentCacheKey(patientID, id)
	cachedAppointment, err := r.cache.Get(ctx, cacheKey)
	if err == nil {
		var appointment models.Appointment
		if err := json.Unmarshal([]byte(cachedAppointment), &appointment); err == nil {
			return &appointment, nil
		}
	} else if err != redis.Nil {
		log.Printf("Failed to get appointment from cache: %v", err)
	}

	var appointment models.Appointment
	err = database.DB.Select("id, patient_id, doctor_id, date_time, created_at, status").
		Preload("Patient", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, first_name, last_name")
		}).
		Preload("Doctor", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, first_name, last_name")
		}).
		First(&appointment, "id = ? AND patient_id = ?", id, patientID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get appointment: %w", err)
	}

	appointmentJSON, err := json.Marshal(appointment)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal appointment: %w", err)
	}
	if err := r.cache.Set(ctx, cacheKey, appointmentJSON, AppointmentCacheExpiry); err != nil {
		log.Printf("Failed to set appointment in cache: %v", err)
	}

	return &appointment, nil
}

func (r *AppointmentRepository) GetAll(ctx context.Context) ([]models.Appointment, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	cacheKey := "appointments_cache"
	cachedAppointments, err := r.cache.Get(ctx, cacheKey)
	if err == nil {
		var appointments []models.Appointment
		if err := json.Unmarshal([]byte(cachedAppointments), &appointments); err == nil {
			return appointments, nil
		}
	} else if err != redis.Nil {
		log.Printf("Failed to get appointments from cache: %v", err)
	}

	var appointments []models.Appointment
	err = database.DB.Select("id, patient_id, doctor_id, date_time, created_at, status").
		Preload("Patient", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, first_name, last_name")
		}).
		Preload("Doctor", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, first_name, last_name")
		}).
		Order("created_at DESC").
		Find(&appointments).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get all appointments: %w", err)
	}

	appointmentsJSON, err := json.Marshal(appointments)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal appointments: %w", err)
	}
	if err := r.cache.Set(ctx, cacheKey, appointmentsJSON, AppointmentCacheExpiry); err != nil {
		log.Printf("Failed to set appointments in cache: %v", err)
	}

	return appointments, nil
}

func (r *AppointmentRepository) Update(ctx context.Context, appointment *models.Appointment) error {
	lockKey := fmt.Sprintf("appointment_lock:%s_%d", appointment.PatientID, appointment.ID)
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

	// Validate the Status field
	if appointment.Status != "scheduled" && appointment.Status != "fulfilled" && appointment.Status != "cancelled" {
		return errors.New("invalid status value")
	}

	err = database.DB.Save(appointment).Error
	if err != nil {
		return fmt.Errorf("failed to update appointment: %w", err)
	}
	if err := r.cache.Delete(ctx, r.getAppointmentCacheKey(appointment.PatientID, appointment.ID)); err != nil {
		return fmt.Errorf("failed to delete appointment cache: %w", err)
	}
	if err := r.cache.DeleteAll(ctx, "appointments_cache"); err != nil {
		return fmt.Errorf("failed to delete all appointments cache: %w", err)
	}
	// Invalidate the specific patient cache and all appointments cache
	if err := r.cache.Delete(ctx, r.getPatientCacheKey(appointment.PatientID)); err != nil {
		return fmt.Errorf("failed to delete patient cache: %w", err)
	}
	return r.cache.DeleteAll(ctx, "patients_cache")
}

func (r *AppointmentRepository) Delete(ctx context.Context, patientID string, id uint) error {
	lockKey := fmt.Sprintf("appointment_lock:%s_%d", patientID, id)
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

	err = database.DB.Delete(&models.Appointment{}, "id = ? AND patient_id = ?", id, patientID).Error
	if err != nil {
		return fmt.Errorf("failed to delete appointment: %w", err)
	}
	if err := r.cache.Delete(ctx, r.getAppointmentCacheKey(patientID, id)); err != nil {
		return fmt.Errorf("failed to delete appointment cache: %w", err)
	}
	if err := r.cache.DeleteAll(ctx, "appointments_cache"); err != nil {
		return fmt.Errorf("failed to delete all appointments cache: %w", err)
	}
	// Invalidate the specific patient cache and all appointments cache
	if err := r.cache.Delete(ctx, r.getPatientCacheKey(patientID)); err != nil {
		return fmt.Errorf("failed to delete patient cache: %w", err)
	}
	return r.cache.DeleteAll(ctx, "patients_cache")
}

func (r *AppointmentRepository) DeleteCache(ctx context.Context, patientID string, id uint) error {
	return r.cache.Delete(ctx, r.getAppointmentCacheKey(patientID, id))
}

func (r *AppointmentRepository) DeleteAllCache(ctx context.Context) error {
	return r.cache.DeleteAll(ctx, "appointments_cache")
}

func (r *AppointmentRepository) getAppointmentCacheKey(patientID string, id uint) string {
	return fmt.Sprintf("appointment_cache:%s_%d", patientID, id)
}

func (r *AppointmentRepository) getPatientCacheKey(patientID string) string {
	return fmt.Sprintf("patient_cache:%s", patientID)
}
