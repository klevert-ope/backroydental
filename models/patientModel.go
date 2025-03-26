package models

import (
	"time"
)

// Doctor model
type Doctor struct {
	ID           string        `gorm:"primaryKey;column:id" json:"id"`
	FirstName    string        `gorm:"column:first_name;not null" json:"first_name"`
	LastName     string        `gorm:"column:last_name;not null;index" json:"last_name"`
	CreatedAt    time.Time     `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	Appointments []Appointment `gorm:"foreignKey:DoctorID;references:ID" json:"-"`
	Billings     []Billing     `gorm:"foreignKey:DoctorID;references:ID" json:"-"`
}

func (Doctor) TableName() string {
	return "doctor"
}

// Patient model
type Patient struct {
	ID                string             `gorm:"primaryKey;column:id" json:"id"`
	FirstName         string             `gorm:"column:first_name;not null" json:"first_name"`
	MiddleName        string             `gorm:"column:middle_name" json:"middle_name"`
	LastName          string             `gorm:"column:last_name;not null;index" json:"last_name"`
	Sex               string             `gorm:"column:sex;check:sex IN ('Male', 'Female', 'Other');not null" json:"sex"`
	DateOfBirth       string             `gorm:"column:date_of_birth;not null;index" json:"date_of_birth"`
	Insured           bool               `gorm:"column:insured;not null" json:"insured"`
	Cash              bool               `gorm:"column:cash;not null" json:"cash"`
	InsuranceCompany  string             `gorm:"column:insurance_company" json:"insurance_company"`
	Scheme            string             `gorm:"column:scheme" json:"scheme"`
	CoverLimit        float64            `gorm:"column:cover_limit" json:"cover_limit"`
	Occupation        string             `gorm:"column:occupation" json:"occupation"`
	PlaceOfWork       string             `gorm:"column:place_of_work" json:"place_of_work"`
	Phone             string             `gorm:"column:phone" json:"phone"`
	Email             string             `gorm:"column:email" json:"email"`
	Address           string             `gorm:"column:address" json:"address"`
	CreatedAt         time.Time          `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	EmergencyContacts []EmergencyContact `gorm:"foreignKey:PatientID;references:ID" json:"-"`
	Examinations      []Examination      `gorm:"foreignKey:PatientID;references:ID" json:"-"`
	Billings          []Billing          `gorm:"foreignKey:PatientID;references:ID" json:"-"`
	TreatmentPlans    []TreatmentPlan    `gorm:"foreignKey:PatientID;references:ID" json:"-"`
	Appointments      []Appointment      `gorm:"foreignKey:PatientID;references:ID" json:"-"`
}

func (Patient) TableName() string {
	return "patient"
}

// EmergencyContact model
type EmergencyContact struct {
	ID           uint    `gorm:"primaryKey;autoIncrement;column:id;index" json:"id"`
	PatientID    string  `gorm:"column:patient_id;not null;index;uniqueIndex:idx_patient_phone" json:"patient_id"`
	Name         string  `gorm:"column:name;not null" json:"name"`
	Phone        string  `gorm:"column:phone;not null;uniqueIndex:idx_patient_phone" json:"phone"`
	Relationship string  `gorm:"column:relationship;not null" json:"relationship"`
	Patient      Patient `gorm:"foreignKey:PatientID;references:ID" json:"-"`
}

func (EmergencyContact) TableName() string {
	return "emergency_contact"
}

// InsuranceCompany model
type InsuranceCompany struct {
	ID   string `gorm:"primaryKey;column:id" json:"id"`
	Name string `gorm:"column:name;unique;not null" json:"name"`
}

func (InsuranceCompany) TableName() string {
	return "insurance_company"
}

// Examination model
type Examination struct {
	ID        uint      `gorm:"primaryKey;autoIncrement;column:id;index" json:"id"`
	PatientID string    `gorm:"column:patient_id;not null;index" json:"patient_id"`
	Report    string    `gorm:"column:report;not null" json:"report"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	Patient   Patient   `gorm:"foreignKey:PatientID;references:ID" json:"-"`
}

func (Examination) TableName() string {
	return "examination"
}

// Billing model
type Billing struct {
	BillingID           string    `gorm:"primaryKey;column:billing_id" json:"billing_id"`
	PatientID           string    `gorm:"column:patient_id;not null;index" json:"patient_id"`
	DoctorID            string    `gorm:"column:doctor_id;not null;index" json:"doctor_id"`
	Procedure           string    `gorm:"column:procedure;not null" json:"procedure"`
	BillingAmount       float64   `gorm:"column:billing_amount;not null" json:"billing_amount"`
	PaidCashAmount      float64   `gorm:"column:paid_cash_amount" json:"paid_cash_amount"`
	PaidInsuranceAmount float64   `gorm:"column:paid_insurance_amount" json:"paid_insurance_amount"`
	Balance             float64   `gorm:"column:balance" json:"balance"`
	TotalReceived       float64   `gorm:"column:total_received" json:"total_received"`
	CreatedAt           time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	Patient             Patient   `gorm:"foreignKey:PatientID;references:ID" json:"-"`
	Doctor              Doctor    `gorm:"foreignKey:DoctorID;references:ID" json:"-"`
}

func (Billing) TableName() string {
	return "billing"
}

// TreatmentPlan model
type TreatmentPlan struct {
	ID        uint      `gorm:"primaryKey;autoIncrement;column:id;index" json:"id"`
	PatientID string    `gorm:"column:patient_id;not null;index" json:"patient_id"`
	Plan      string    `gorm:"column:plan;not null" json:"plan"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	Patient   Patient   `gorm:"foreignKey:PatientID;references:ID" json:"-"`
}

func (TreatmentPlan) TableName() string {
	return "treatment_plan"
}

// Appointment model
type Appointment struct {
	ID        uint      `gorm:"primaryKey;autoIncrement;column:id;index" json:"id"`
	PatientID string    `gorm:"column:patient_id;not null;index" json:"patient_id"`
	DoctorID  string    `gorm:"column:doctor_id;not null;index" json:"doctor_id"`
	DateTime  string    `gorm:"column:date_time;not null;index" json:"date_time"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	Status    string    `gorm:"column:status;check:status IN ('scheduled', 'fulfilled', 'cancelled');not null" json:"status"`
	Patient   Patient   `gorm:"foreignKey:PatientID;references:ID" json:"patient"`
	Doctor    Doctor    `gorm:"foreignKey:DoctorID;references:ID" json:"doctor"`
}

func (Appointment) TableName() string {
	return "appointment"
}
