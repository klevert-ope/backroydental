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
	"gorm.io/gorm/clause"
)

const (
	EmergencyContactCacheExpiry = 7 * 24 * time.Hour
)

type EmergencyContactRepository struct {
	cache *cache.Cache
}

func NewEmergencyContactRepository(cache *cache.Cache) *EmergencyContactRepository {
	return &EmergencyContactRepository{cache: cache}
}

func (r *EmergencyContactRepository) Create(ctx context.Context, contact *models.EmergencyContact) error {
	lockKey := fmt.Sprintf("emergency_contact_lock:%s", contact.PatientID)
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

	// Insert the emergency contact record if it does not exist
	err = database.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "patient_id"}, {Name: "phone"}},
		DoUpdates: clause.AssignmentColumns([]string{"name", "relationship"}),
	}).Create(contact).Error
	if err != nil {
		return fmt.Errorf("failed to create emergency contact: %w", err)
	}

	// Delete cache for the newly created emergency contact and all emergency contacts
	if err := r.cache.Delete(ctx, r.getEmergencyContactCacheKey(contact.PatientID, contact.ID)); err != nil {
		return fmt.Errorf("failed to delete emergency contact cache: %w", err)
	}
	if err := r.cache.DeleteAll(ctx, "emergency_contacts_cache"); err != nil {
		return fmt.Errorf("failed to delete all emergency contacts cache: %w", err)
	}
	// Invalidate the specific patient cache and all emergency contacts cache
	if err := r.cache.Delete(ctx, r.getPatientCacheKey(contact.PatientID)); err != nil {
		return fmt.Errorf("failed to delete patient cache: %w", err)
	}
	return r.cache.DeleteAll(ctx, "patients_cache")
}

func (r *EmergencyContactRepository) Update(ctx context.Context, contact *models.EmergencyContact) error {
	// Acquire a lock based on the contact ID and patient ID
	lockKey := fmt.Sprintf("emergency_contact_lock:%s_%d", contact.PatientID, contact.ID)
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

	// Fetch the existing contact to check if it exists
	existingContact, err := r.GetByID(ctx, contact.PatientID, contact.ID)
	if err != nil {
		return fmt.Errorf("failed to get existing emergency contact: %w", err)
	}
	if existingContact == nil {
		return errors.New("emergency contact not found")
	}

	// Update the contact details
	existingContact.Name = contact.Name
	existingContact.Relationship = contact.Relationship
	existingContact.Phone = contact.Phone

	// Save the updated contact to the database
	err = database.DB.Save(existingContact).Error
	if err != nil {
		return fmt.Errorf("failed to update emergency contact: %w", err)
	}

	// Delete cache for the updated emergency contact and all emergency contacts
	if err := r.cache.Delete(ctx, r.getEmergencyContactCacheKey(contact.PatientID, contact.ID)); err != nil {
		return fmt.Errorf("failed to delete emergency contact cache: %w", err)
	}
	if err := r.cache.DeleteAll(ctx, "emergency_contacts_cache"); err != nil {
		return fmt.Errorf("failed to delete all emergency contacts cache: %w", err)
	}
	// Invalidate the specific patient cache and all emergency contacts cache
	if err := r.cache.Delete(ctx, r.getPatientCacheKey(contact.PatientID)); err != nil {
		return fmt.Errorf("failed to delete patient cache: %w", err)
	}
	return r.cache.DeleteAll(ctx, "patients_cache")
}

func (r *EmergencyContactRepository) GetByID(ctx context.Context, patientID string, id uint) (*models.EmergencyContact, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	cacheKey := r.getEmergencyContactCacheKey(patientID, id)
	cachedContact, err := r.cache.Get(ctx, cacheKey)
	if err == nil {
		var contact models.EmergencyContact
		if err := json.Unmarshal([]byte(cachedContact), &contact); err != nil {
			log.Printf("Failed to unmarshal emergency contact from cache: %v", err)
		} else {
			return &contact, nil
		}
	} else if err != redis.Nil {
		log.Printf("Failed to get emergency contact from cache: %v", err)
	}

	var contact models.EmergencyContact
	err = database.DB.Select("id, patient_id, name, phone, relationship").
		Preload("Patient", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, first_name, last_name")
		}).
		First(&contact, "patient_id = ? AND id = ?", patientID, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get emergency contact: %w", err)
	}

	contactJSON, err := json.Marshal(contact)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal emergency contact: %w", err)
	}
	if err := r.cache.Set(ctx, cacheKey, contactJSON, EmergencyContactCacheExpiry); err != nil {
		log.Printf("Failed to set emergency contact in cache: %v", err)
	}

	return &contact, nil
}

func (r *EmergencyContactRepository) GetAll(ctx context.Context) ([]models.EmergencyContact, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	cacheKey := "emergency_contacts_cache"
	cachedContacts, err := r.cache.Get(ctx, cacheKey)
	if err == nil {
		var contacts []models.EmergencyContact
		if err := json.Unmarshal([]byte(cachedContacts), &contacts); err == nil {
			return contacts, nil
		}
	} else if err != redis.Nil {
		log.Printf("Failed to get emergency contacts from cache: %v", err)
	}

	var contacts []models.EmergencyContact
	err = database.DB.Select("id, patient_id, name, phone, relationship").
		Preload("Patient", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, first_name, last_name")
		}).
		Find(&contacts).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get all emergency contacts: %w", err)
	}

	contactsJSON, err := json.Marshal(contacts)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal emergency contacts: %w", err)
	}
	if err := r.cache.Set(ctx, cacheKey, contactsJSON, EmergencyContactCacheExpiry); err != nil {
		log.Printf("Failed to set emergency contacts in cache: %v", err)
	}

	return contacts, nil
}

func (r *EmergencyContactRepository) Delete(ctx context.Context, patientID string, id uint) error {
	lockKey := fmt.Sprintf("emergency_contact_lock:%s_%d", patientID, id)
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

	err = database.DB.Delete(&models.EmergencyContact{}, "patient_id = ? AND id = ?", patientID, id).Error
	if err != nil {
		return fmt.Errorf("failed to delete emergency contact: %w", err)
	}
	// Delete cache for the deleted emergency contact and all emergency contacts
	if err := r.cache.Delete(ctx, r.getEmergencyContactCacheKey(patientID, id)); err != nil {
		return fmt.Errorf("failed to delete emergency contact cache: %w", err)
	}
	if err := r.cache.DeleteAll(ctx, "emergency_contacts_cache"); err != nil {
		return fmt.Errorf("failed to delete all emergency contacts cache: %w", err)
	}
	// Invalidate the specific patient cache and all emergency contacts cache
	if err := r.cache.Delete(ctx, r.getPatientCacheKey(patientID)); err != nil {
		return fmt.Errorf("failed to delete patient cache: %w", err)
	}
	return r.cache.DeleteAll(ctx, "patients_cache")
}

func (r *EmergencyContactRepository) DeleteCache(ctx context.Context, patientID string, id uint) error {
	return r.cache.Delete(ctx, r.getEmergencyContactCacheKey(patientID, id))
}

func (r *EmergencyContactRepository) DeleteAllCache(ctx context.Context) error {
	return r.cache.DeleteAll(ctx, "emergency_contacts_cache")
}

func (r *EmergencyContactRepository) getEmergencyContactCacheKey(patientID string, id uint) string {
	return fmt.Sprintf("emergency_contact_cache:%s_%d", patientID, id)
}

func (r *EmergencyContactRepository) getPatientCacheKey(patientID string) string {
	return fmt.Sprintf("patient_cache:%s", patientID)
}
