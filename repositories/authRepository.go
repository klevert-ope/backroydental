package repositories

import (
	"RoyDental/cache"
	"RoyDental/models"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"
)

const (
	UserCacheExpiry = 7 * 24 * time.Hour
)

type UserRepository interface {
	EmailExists(ctx context.Context, email string) (bool, error)
	GetUserByUsername(ctx context.Context, username string) (*models.User, error)
	GetUserByEmail(ctx context.Context, email string) (*models.User, error)
	CreateUser(ctx context.Context, user *models.User) error
	AuthenticateUser(ctx context.Context, username, password string) (*models.User, error)
	ValidateRoleID(ctx context.Context, roleID int64) error
	UpdateUserEmail(ctx context.Context, userID int64, newEmail string) error
	UpdateUserPassword(ctx context.Context, userID int64, hashedPassword string) error
	GetAllUsers(ctx context.Context) ([]models.User, error)
	DeleteUserCache(ctx context.Context, identifier string) error
	GetUserByID(ctx context.Context, userID int64) (*models.User, error)
	UpdateUserProfile(ctx context.Context, userID int64, username, email string) error
	GetUserPermissions(ctx context.Context, userID int64) ([]models.Permission, error)
	DeleteUser(ctx context.Context, userID int64) error
}

type userRepository struct {
	db    *gorm.DB
	cache *cache.Cache
}

func NewUserRepository(db *gorm.DB, cache *cache.Cache) UserRepository {
	return &userRepository{db: db, cache: cache}
}

func (r *userRepository) EmailExists(ctx context.Context, email string) (bool, error) {
	var count int64
	err := r.db.Model(&models.User{}).Where("email = ?", email).Count(&count).Error
	if err != nil {
		return false, fmt.Errorf("failed to check email existence: %w", err)
	}
	return count > 0, nil
}

func (r *userRepository) GetUserByUsername(ctx context.Context, username string) (*models.User, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	cacheKey := r.getUserCacheKey(username)
	cachedUser, err := r.cache.Get(ctx, cacheKey)
	if err == nil {
		var user models.User
		if err := json.Unmarshal([]byte(cachedUser), &user); err == nil {
			return &user, nil
		}
	} else if err != redis.Nil {
		log.Printf("Failed to get user from cache: %v", err)
	}

	var user models.User
	err = r.db.Select("id, username, email, role_id, created_at").
		Preload("Role", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, name, description")
		}).
		Where("username = ?", username).
		First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	userJSON, err := json.Marshal(user)
	if err != nil {
		return nil, err
	}
	if err := r.cache.Set(ctx, cacheKey, userJSON, UserCacheExpiry); err != nil {
		log.Printf("Failed to set user in cache: %v", err)
	}

	return &user, nil
}

func (r *userRepository) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	cacheKey := r.getUserCacheKey(email)
	cachedUser, err := r.cache.Get(ctx, cacheKey)
	if err == nil {
		var user models.User
		if err := json.Unmarshal([]byte(cachedUser), &user); err == nil {
			return &user, nil
		}
	} else if err != redis.Nil {
		log.Printf("Failed to get user from cache: %v", err)
	}

	var user models.User
	err = r.db.Select("id, username, email, role_id, created_at").
		Preload("Role", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, name, description")
		}).
		Where("email = ?", email).
		First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	userJSON, err := json.Marshal(user)
	if err != nil {
		return nil, err
	}
	if err := r.cache.Set(ctx, cacheKey, userJSON, UserCacheExpiry); err != nil {
		log.Printf("Failed to set user in cache: %v", err)
	}

	return &user, nil
}

func (r *userRepository) CreateUser(ctx context.Context, user *models.User) error {
	return r.db.Create(&user).Error
}

func (r *userRepository) AuthenticateUser(ctx context.Context, email, password string) (*models.User, error) {
	var user models.User
	err := r.db.Select("id, username, email, password, role_id, created_at").
		Preload("Role", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, name, description")
		}).
		Where("email = ?", email).
		First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("invalid email or password")
		}
		return nil, err
	}

	return &user, nil
}

func (r *userRepository) ValidateRoleID(ctx context.Context, roleID int64) error {
	var count int64
	err := r.db.Model(&models.Role{}).Where("id = ?", roleID).Count(&count).Error
	if err != nil {
		return fmt.Errorf("failed to validate role ID: %w", err)
	}
	return nil
}

func (r *userRepository) UpdateUserEmail(ctx context.Context, userID int64, newEmail string) error {
	return r.db.Model(&models.User{}).Where("id = ?", userID).Update("email", newEmail).Error
}

func (r *userRepository) UpdateUserPassword(ctx context.Context, userID int64, hashedPassword string) error {
	return r.db.Model(&models.User{}).Where("id = ?", userID).Update("password", hashedPassword).Error
}

func (r *userRepository) GetAllUsers(ctx context.Context) ([]models.User, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var users []models.User
	err := r.db.Select("id, username, email, role_id, created_at").
		Preload("Role", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, name, description")
		}).
		Find(&users).Error
	if err != nil {
		return nil, err
	}
	return users, nil
}

func (r *userRepository) DeleteUserCache(ctx context.Context, identifier string) error {
	cacheKey := r.getUserCacheKey(identifier)
	return r.cache.Delete(ctx, cacheKey)
}

func (r *userRepository) GetUserByID(ctx context.Context, userID int64) (*models.User, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	cacheKey := r.getUserCacheKey(fmt.Sprintf("%d", userID))
	cachedUser, err := r.cache.Get(ctx, cacheKey)
	if err == nil {
		var user models.User
		if err := json.Unmarshal([]byte(cachedUser), &user); err == nil {
			return &user, nil
		}
	} else if err != redis.Nil {
		log.Printf("Failed to get user from cache: %v", err)
	}

	var user models.User
	err = r.db.Select("id, username, email, role_id, created_at").
		Preload("Role", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, name, description")
		}).
		First(&user, userID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	userJSON, err := json.Marshal(user)
	if err != nil {
		return nil, err
	}
	if err := r.cache.Set(ctx, cacheKey, userJSON, UserCacheExpiry); err != nil {
		log.Printf("Failed to set user in cache: %v", err)
	}

	return &user, nil
}

func (r *userRepository) UpdateUserProfile(ctx context.Context, userID int64, username, email string) error {
	return r.db.Model(&models.User{}).Where("id = ?", userID).Updates(map[string]interface{}{
		"username": username,
		"email":    email,
	}).Error
}

func (r *userRepository) GetUserPermissions(ctx context.Context, userID int64) ([]models.Permission, error) {
	var permissions []models.Permission
	err := r.db.Joins("JOIN role_permissions rp ON permissions.id = rp.permission_id").
		Joins("JOIN roles r ON rp.role_id = r.id").
		Where("r.id = (SELECT role_id FROM users WHERE id = ?)", userID).
		Find(&permissions).Error
	if err != nil {
		return nil, err
	}
	return permissions, nil
}

func (r *userRepository) DeleteUser(ctx context.Context, userID int64) error {
	return r.db.Delete(&models.User{}, userID).Error
}

func (r *userRepository) getUserCacheKey(identifier string) string {
	return fmt.Sprintf("user_cache:%s", identifier)
}
