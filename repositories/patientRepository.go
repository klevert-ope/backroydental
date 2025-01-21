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
	PatientCacheExpiry = 7 * 24 * time.Hour
)

type PatientRepository struct {
	cache                *cache.Cache
	emergencyContactRepo *EmergencyContactRepository
	billingRepo          *BillingRepository
	examinationRepo      *ExaminationRepository
	treatmentPlanRepo    *TreatmentPlanRepository
	appointmentRepo      *AppointmentRepository
}

func NewPatientRepository(
	cache *cache.Cache,
	emergencyContactRepo *EmergencyContactRepository,
	billingRepo *BillingRepository,
	examinationRepo *ExaminationRepository,
	treatmentPlanRepo *TreatmentPlanRepository,
	appointmentRepo *AppointmentRepository,
) *PatientRepository {
	return &PatientRepository{
		cache:                cache,
		emergencyContactRepo: emergencyContactRepo,
		billingRepo:          billingRepo,
		examinationRepo:      examinationRepo,
		treatmentPlanRepo:    treatmentPlanRepo,
		appointmentRepo:      appointmentRepo,
	}
}

func (r *PatientRepository) Create(ctx context.Context, patient *models.Patient) error {
	// Handle empty middle name
	middleName := patient.MiddleName
	if middleName == "" {
		middleName = "N/A"
	}

	lockKey := fmt.Sprintf("patient_lock:%s_%s_%s_%s", patient.FirstName, middleName, patient.LastName, patient.DateOfBirth)
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

	// Ensure lock release
	defer func() {
		if err := database.ReleaseLock(ctx, lockKey, lockValue); err != nil {
			log.Printf("Failed to release lock: %v", err)
		}
	}()

	// Check if a record with the same unique fields already exists
	var existingPatient models.Patient
	if err := database.DB.Where("first_name = ? AND middle_name = ? AND last_name = ? AND date_of_birth = ?",
		patient.FirstName, middleName, patient.LastName, patient.DateOfBirth).First(&existingPatient).Error; err == nil {
		return fmt.Errorf("patient with the same details already exists")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("failed to check for existing patient: %w", err)
	}

	// Obtain the next sequence value
	var nextID string
	if err := database.DB.Raw("SELECT 'DP-' || LPAD(nextval('patient_id_seq')::TEXT, 6, '0')").Scan(&nextID).Error; err != nil {
		return fmt.Errorf("failed to obtain next sequence value: %w", err)
	}

	// Assign ID to the patient
	patient.ID = nextID

	// Transaction to create patient and invalidate cache
	return database.DB.Transaction(func(tx *gorm.DB) error {
		// Create the patient record
		if err := tx.Create(patient).Error; err != nil {
			// Rollback sequence in case of failure
			if rollbackErr := tx.Exec("SELECT setval('patient_id_seq', (SELECT last_value FROM patient_id_seq) - 1, false)").Error; rollbackErr != nil {
				return fmt.Errorf("transaction failed and sequence rollback failed: %v, rollback error: %v", err, rollbackErr)
			}
			return fmt.Errorf("failed to create patient: %w", err)
		}

		// Invalidate cache
		if err := r.cache.Delete(ctx, r.getPatientCacheKey(patient.ID)); err != nil {
			return fmt.Errorf("failed to delete patient cache: %w", err)
		}
		return r.cache.DeleteAll(ctx, "patients_cache")
	})
}

func (r *PatientRepository) GetByID(ctx context.Context, id string) (*models.Patient, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	cacheKey := r.getPatientCacheKey(id)
	cachedPatient, err := r.cache.Get(ctx, cacheKey)
	if err == nil {
		var patient models.Patient
		if err := json.Unmarshal([]byte(cachedPatient), &patient); err == nil {
			return &patient, nil
		}
	} else if err != redis.Nil {
		log.Printf("Failed to get patient from cache: %v", err)
	}

	var patient models.Patient
	err = database.DB.Select("id, first_name, middle_name, last_name, sex, date_of_birth, insured, cash, insurance_company, scheme, cover_limit, occupation, place_of_work, phone, email, address, created_at").
		Preload("EmergencyContacts", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, patient_id, name, phone, relationship")
		}).
		Preload("Examinations", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, patient_id, report, created_at")
		}).
		Preload("Billings", func(db *gorm.DB) *gorm.DB {
			return db.Select("billing_id, patient_id, doctor_id, procedure, billing_amount, paid_cash_amount, paid_insurance_amount, balance, total_received, created_at")
		}).
		Preload("TreatmentPlans", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, patient_id, plan, created_at")
		}).
		Preload("Appointments", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, patient_id, doctor_id, date_time, created_at, status")
		}).
		First(&patient, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get patient: %w", err)
	}

	patientJSON, err := json.Marshal(patient)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal patient: %w", err)
	}
	if err := r.cache.Set(ctx, cacheKey, patientJSON, PatientCacheExpiry); err != nil {
		log.Printf("Failed to set patient in cache: %v", err)
	}

	return &patient, nil
}

func (r *PatientRepository) GetAll(ctx context.Context) ([]models.Patient, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	cacheKey := "patients_cache"
	cachedPatients, err := r.cache.Get(ctx, cacheKey)
	if err == nil {
		var patients []models.Patient
		if err := json.Unmarshal([]byte(cachedPatients), &patients); err == nil {
			return patients, nil
		}
	} else if err != redis.Nil {
		log.Printf("Failed to get patients from cache: %v", err)
	}

	var patients []models.Patient
	err = database.DB.Select("id, first_name, middle_name, last_name, sex, date_of_birth, insured, cash, insurance_company, scheme, cover_limit, occupation, place_of_work, phone, email, address, created_at").
		Preload("EmergencyContacts", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, patient_id, name, phone, relationship")
		}).
		Preload("Examinations", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, patient_id, report, created_at")
		}).
		Preload("Billings", func(db *gorm.DB) *gorm.DB {
			return db.Select("billing_id, patient_id, doctor_id, procedure, billing_amount, paid_cash_amount, paid_insurance_amount, balance, total_received, created_at")
		}).
		Preload("TreatmentPlans", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, patient_id, plan, created_at")
		}).
		Preload("Appointments", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, patient_id, doctor_id, date_time, created_at, status")
		}).
		Order("created_at DESC").
		Find(&patients).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get all patients: %w", err)
	}

	patientsJSON, err := json.Marshal(patients)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal patients: %w", err)
	}
	if err := r.cache.Set(ctx, cacheKey, patientsJSON, PatientCacheExpiry); err != nil {
		log.Printf("Failed to set patients in cache: %v", err)
	}

	return patients, nil
}

func (r *PatientRepository) Update(ctx context.Context, patient *models.Patient) error {
	lockKey := fmt.Sprintf("patient_lock:%s", patient.ID)
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

	// Use ON CONFLICT to handle conflicts
	err = database.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"first_name", "middle_name", "last_name", "date_of_birth", "sex", "insured", "cash", "insurance_company", "scheme", "cover_limit", "occupation", "place_of_work", "phone", "email", "address"}),
	}).Save(patient).Error
	if err != nil {
		return fmt.Errorf("failed to update patient: %w", err)
	}

	// Invalidate cache for the updated patient and all patients
	if err := r.cache.Delete(ctx, r.getPatientCacheKey(patient.ID)); err != nil {
		return fmt.Errorf("failed to delete patient cache: %w", err)
	}
	return r.cache.DeleteAll(ctx, "patients_cache")
}

func (r *PatientRepository) Delete(ctx context.Context, id string) error {
	lockKey := fmt.Sprintf("patient_lock:%s", id)
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

	err = database.DB.Delete(&models.Patient{}, "id = ?", id).Error
	if err != nil {
		return fmt.Errorf("failed to delete patient: %w", err)
	}
	// Invalidate cache for the deleted patient and all patients
	if err := r.cache.Delete(ctx, r.getPatientCacheKey(id)); err != nil {
		return fmt.Errorf("failed to delete patient cache: %w", err)
	}
	return r.cache.DeleteAll(ctx, "patients_cache")
}

func (r *PatientRepository) DeletePatientAndRelated(ctx context.Context, id string) error {
	lockKey := fmt.Sprintf("patient_lock:%s", id)
	lockValue := uuid.New().String() // Generate a unique lock value
	locked, err := database.NewLock(ctx, lockKey, lockValue, time.Minute)
	if err != nil {
		return fmt.Errorf("failed to acquire lock: %w", err)
	}
	if !locked {
		return errors.New("failed to acquire lock")
	}
	defer func() {
		if err := database.ReleaseLock(ctx, lockKey, lockValue); err != nil {
			log.Printf("Failed to release lock: %v", err)
		}
	}()

	return database.DB.Transaction(func(tx *gorm.DB) error {
		if err := r.invalidateEmergencyContactsCache(ctx, tx, id); err != nil {
			return err
		}

		if err := r.invalidateExaminationsCache(ctx, tx, id); err != nil {
			return err
		}

		if err := r.invalidateBillingsCache(ctx, tx, id); err != nil {
			return err
		}

		if err := r.invalidateTreatmentPlansCache(ctx, tx, id); err != nil {
			return err
		}

		if err := r.invalidateAppointmentsCache(ctx, tx, id); err != nil {
			return err
		}

		if err := tx.Where("patient_id = ?", id).Delete(&models.EmergencyContact{}).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if err := tx.Where("patient_id = ?", id).Delete(&models.Examination{}).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if err := tx.Where("patient_id = ?", id).Delete(&models.Billing{}).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if err := tx.Where("patient_id = ?", id).Delete(&models.TreatmentPlan{}).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if err := tx.Where("patient_id = ?", id).Delete(&models.Appointment{}).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		if err := tx.Delete(&models.Patient{}, "id = ?", id).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		if err := r.cache.Delete(ctx, r.getPatientCacheKey(id)); err != nil {
			return err
		}
		if err := r.cache.DeleteAll(ctx, "patients_cache"); err != nil {
			return err
		}

		if err := r.appointmentRepo.DeleteAllCache(ctx); err != nil {
			return err
		}

		if err := r.emergencyContactRepo.DeleteAllCache(ctx); err != nil {
			return err
		}

		if err := r.billingRepo.DeleteAllCache(ctx); err != nil {
			return err
		}

		if err := r.examinationRepo.DeleteAllCache(ctx); err != nil {
			return err
		}

		if err := r.treatmentPlanRepo.DeleteAllCache(ctx); err != nil {
			return err
		}

		return nil
	})
}

func (r *PatientRepository) invalidateEmergencyContactsCache(ctx context.Context, tx *gorm.DB, patientID string) error {
	var emergencyContacts []models.EmergencyContact
	if err := tx.Where("patient_id = ?", patientID).Find(&emergencyContacts).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}

	for _, contact := range emergencyContacts {
		if err := r.emergencyContactRepo.DeleteCache(ctx, contact.PatientID, contact.ID); err != nil {
			return err
		}
	}
	return nil
}

func (r *PatientRepository) invalidateExaminationsCache(ctx context.Context, tx *gorm.DB, patientID string) error {
	var examinations []models.Examination
	if err := tx.Where("patient_id = ?", patientID).Find(&examinations).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}

	for _, examination := range examinations {
		if err := r.examinationRepo.DeleteCache(ctx, examination.PatientID, examination.ID); err != nil {
			return err
		}
	}
	return nil
}

func (r *PatientRepository) invalidateBillingsCache(ctx context.Context, tx *gorm.DB, patientID string) error {
	var billings []models.Billing
	if err := tx.Where("patient_id = ?", patientID).Find(&billings).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}

	for _, billing := range billings {
		if err := r.billingRepo.DeleteCache(ctx, billing.BillingID); err != nil {
			return err
		}
	}
	return nil
}

func (r *PatientRepository) invalidateTreatmentPlansCache(ctx context.Context, tx *gorm.DB, patientID string) error {
	var treatmentPlans []models.TreatmentPlan
	if err := tx.Where("patient_id = ?", patientID).Find(&treatmentPlans).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}

	for _, treatmentPlan := range treatmentPlans {
		if err := r.treatmentPlanRepo.DeleteCache(ctx, treatmentPlan.PatientID, treatmentPlan.ID); err != nil {
			return err
		}
	}
	return nil
}

func (r *PatientRepository) invalidateAppointmentsCache(ctx context.Context, tx *gorm.DB, patientID string) error {
	var appointments []models.Appointment
	if err := tx.Where("patient_id = ?", patientID).Find(&appointments).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}

	for _, appointment := range appointments {
		if err := r.appointmentRepo.DeleteCache(ctx, appointment.PatientID, appointment.ID); err != nil {
			return err
		}
	}
	return nil
}

func (r *PatientRepository) getPatientCacheKey(patientID string) string {
	return fmt.Sprintf("patient_cache:%s", patientID)
}
