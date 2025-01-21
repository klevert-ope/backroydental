package models

import (
	"time"

	"gorm.io/gorm"
)

// Role represents a user role
type Role struct {
	ID          int64        `gorm:"primaryKey;column:id" json:"id"`
	Name        string       `gorm:"size:50;not null;unique;index;column:name" json:"name"`
	Description string       `gorm:"type:text;column:description" json:"description"`
	CreatedAt   time.Time    `gorm:"autoCreateTime;column:created_at" json:"created_at"`
	Permissions []Permission `gorm:"many2many:role_permissions;" json:"permissions"`
}

func (Role) TableName() string {
	return "roles"
}

// SeedRoles inserts initial roles into the database
func SeedRoles(db *gorm.DB) error {
	initialRoles := []Role{
		{Name: "Admin", Description: "Full access to the system"},
		{Name: "Doctor", Description: "Can manage patients and prescriptions"},
		{Name: "Receptionist", Description: "Can handle appointments and billing"},
		{Name: "Patient", Description: "Limited access to personal data"},
	}
	return db.Transaction(func(tx *gorm.DB) error {
		for _, role := range initialRoles {
			if err := tx.FirstOrCreate(&role, Role{Name: role.Name}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// User represents a user in the system
type User struct {
	ID        int64     `gorm:"primaryKey;column:id" json:"id"`
	Username  string    `gorm:"size:100;not null;unique;index;column:username" json:"username"`
	Email     string    `gorm:"size:255;not null;unique;index;column:email" json:"email"`
	Password  string    `gorm:"size:255;not null;column:password" json:"password"`
	RoleID    int64     `gorm:"index;not null;column:role_id" json:"role_id"`
	Role      Role      `gorm:"foreignKey:RoleID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"role"`
	CreatedAt time.Time `gorm:"autoCreateTime;column:created_at" json:"created_at"`
}

func (User) TableName() string {
	return "users"
}

// Permission represents a permission in the system
type Permission struct {
	ID          int64  `gorm:"primaryKey;column:id" json:"id"`
	Name        string `gorm:"size:100;not null;unique;index;column:name" json:"name"`
	Description string `gorm:"type:text;column:description" json:"description"`
}

func (Permission) TableName() string {
	return "permissions"
}

// SeedPermissions inserts initial permissions into the database
func SeedPermissions(db *gorm.DB) error {
	initialPermissions := []Permission{
		{Name: "manage_users", Description: "Create, update, or delete users"},
		{Name: "view_patients", Description: "View patient data"},
		{Name: "edit_prescriptions", Description: "Edit patient prescriptions"},
		{Name: "manage_appointments", Description: "Create or update appointments"},
		{Name: "view_self", Description: "View personal data"},
	}
	return db.Transaction(func(tx *gorm.DB) error {
		for _, permission := range initialPermissions {
			if err := tx.FirstOrCreate(&permission, Permission{Name: permission.Name}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// RolePermission represents the association between roles and permissions
type RolePermission struct {
	ID           int64 `gorm:"primaryKey;column:id" json:"id"`
	RoleID       int64 `gorm:"index;column:role_id" json:"role_id"`
	PermissionID int64 `gorm:"index;column:permission_id" json:"permission_id"`
}

func (RolePermission) TableName() string {
	return "role_permissions"
}

// SeedRolePermissions inserts initial role permissions into the database
func SeedRolePermissions(db *gorm.DB) error {
	initialRolePermissions := []RolePermission{
		{RoleID: 1, PermissionID: 1}, // Admin: manage_users
		{RoleID: 1, PermissionID: 2}, // Admin: view_patients
		{RoleID: 1, PermissionID: 3}, // Admin: edit_prescriptions
		{RoleID: 1, PermissionID: 4}, // Admin: manage_appointments
		{RoleID: 2, PermissionID: 2}, // Doctor: view_patients
		{RoleID: 2, PermissionID: 3}, // Doctor: edit_prescriptions
		{RoleID: 3, PermissionID: 4}, // Receptionist: manage_appointments
		{RoleID: 4, PermissionID: 5}, // Patient: view_self
	}
	return db.Transaction(func(tx *gorm.DB) error {
		for _, rolePermission := range initialRolePermissions {
			if err := tx.FirstOrCreate(&rolePermission, RolePermission{RoleID: rolePermission.RoleID, PermissionID: rolePermission.PermissionID}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
