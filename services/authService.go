package services

import (
	"RoyDental/database"
	"RoyDental/models"
	"RoyDental/repositories"
	"RoyDental/utils"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
)

const (
	UserCacheExpiry = 7 * 24 * time.Hour
)

type UserService interface {
	ValidateAndCreateUser(ctx context.Context, user *models.User) error
	AuthenticateUser(ctx context.Context, username, password string) (*models.User, error)
	UpdateUserEmail(ctx context.Context, userID int64, newEmail string) error
	UpdateUserPassword(ctx context.Context, userID int64, hashedPassword string) error
	GetAllUsers(ctx context.Context) ([]models.User, error)
	GetUserByID(ctx context.Context, userID int64) (*models.User, error)
	GetUserByUsername(ctx context.Context, username string) (*models.User, error)
	GetUserByEmail(ctx context.Context, email string) (*models.User, error)
	UpdateUserProfile(ctx context.Context, userID int64, username, email string) error
	GetUserPermissions(ctx context.Context, userID int64) ([]models.Permission, error)
	DeleteUser(ctx context.Context, userID int64) error
}

type userService struct {
	userRepo repositories.UserRepository
}

func NewUserService(userRepo repositories.UserRepository) UserService {
	return &userService{userRepo: userRepo}
}

func (s *userService) ValidateAndCreateUser(ctx context.Context, user *models.User) error {
	lockKey := fmt.Sprintf("user_lock:%s", user.Email)
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

	// Validate user data before creating
	if err := utils.ValidateUserData(*user); err != nil {
		return fmt.Errorf("invalid user data: %w", err)
	}

	if user.Password == "" {
		return errors.New("password cannot be blank")
	}

	if exists, err := s.userRepo.EmailExists(ctx, user.Email); err != nil || exists {
		return errors.New("email already registered")
	}

	if err := s.userRepo.ValidateRoleID(ctx, user.RoleID); err != nil {
		return fmt.Errorf("invalid role ID: %w", err)
	}

	hashedPassword, err := utils.HashPassword(user.Password)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}
	user.Password = hashedPassword

	return s.userRepo.CreateUser(ctx, user)
}

func (s *userService) AuthenticateUser(ctx context.Context, email, password string) (*models.User, error) {
	user, err := s.userRepo.AuthenticateUser(ctx, email, password)
	if err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}

	if !utils.CheckPassword(user.Password, password) {
		return nil, errors.New("invalid email or password")
	}

	// Cache the user data on successful login
	userJSON, err := json.Marshal(user)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal user data: %w", err)
	}
	cacheKey := fmt.Sprintf("user_cache:%s", email)
	if err := database.RedisClient.Set(ctx, cacheKey, userJSON, UserCacheExpiry).Err(); err != nil {
		log.Printf("Failed to set user in cache: %v", err)
	}

	return user, nil
}

func (s *userService) UpdateUserEmail(ctx context.Context, userID int64, newEmail string) error {
	lockKey := fmt.Sprintf("user_lock:%d", userID)
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

	if err := s.userRepo.UpdateUserEmail(ctx, userID, newEmail); err != nil {
		return fmt.Errorf("failed to update user email: %w", err)
	}

	// Invalidate cache for both old and new email
	if err := s.userRepo.DeleteUserCache(ctx, newEmail); err != nil {
		return fmt.Errorf("failed to delete user cache: %w", err)
	}
	user, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user by ID: %w", err)
	}
	if user == nil {
		return errors.New("user not found")
	}
	return s.userRepo.DeleteUserCache(ctx, user.Email)
}

func (s *userService) UpdateUserPassword(ctx context.Context, userID int64, hashedPassword string) error {
	lockKey := fmt.Sprintf("user_lock:%d", userID)
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

	if err := s.userRepo.UpdateUserPassword(ctx, userID, hashedPassword); err != nil {
		return fmt.Errorf("failed to update user password: %w", err)
	}

	user, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user by ID: %w", err)
	}
	if user == nil {
		return errors.New("user not found")
	}

	// Invalidate cache for the user
	return s.userRepo.DeleteUserCache(ctx, user.Username)
}

func (s *userService) GetAllUsers(ctx context.Context) ([]models.User, error) {
	return s.userRepo.GetAllUsers(ctx)
}

func (s *userService) GetUserByID(ctx context.Context, userID int64) (*models.User, error) {
	return s.userRepo.GetUserByID(ctx, userID)
}

func (s *userService) GetUserByUsername(ctx context.Context, username string) (*models.User, error) {
	return s.userRepo.GetUserByUsername(ctx, username)
}

func (s *userService) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	return s.userRepo.GetUserByEmail(ctx, email)
}

func (s *userService) UpdateUserProfile(ctx context.Context, userID int64, username, email string) error {
	lockKey := fmt.Sprintf("user_lock:%d", userID)
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

	if err := s.userRepo.UpdateUserProfile(ctx, userID, username, email); err != nil {
		return fmt.Errorf("failed to update user profile: %w", err)
	}

	// Invalidate cache for the user
	return s.userRepo.DeleteUserCache(ctx, username)
}

func (s *userService) GetUserPermissions(ctx context.Context, userID int64) ([]models.Permission, error) {
	return s.userRepo.GetUserPermissions(ctx, userID)
}

func (s *userService) DeleteUser(ctx context.Context, userID int64) error {
	lockKey := fmt.Sprintf("user_lock:%d", userID)
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

	user, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user by ID: %w", err)
	}
	if user == nil {
		return errors.New("user not found")
	}

	// Invalidate cache for the user
	if err := s.userRepo.DeleteUserCache(ctx, user.Username); err != nil {
		return fmt.Errorf("failed to delete user cache: %w", err)
	}

	return s.userRepo.DeleteUser(ctx, userID)
}
