package utils

import (
	"RoyDental/cache"
	"context"
	"fmt"
	"math/rand"
	"time"
)

// GenerateResetCode generates a random 6-digit reset code.
func GenerateResetCode() string {
	rand.Seed(time.Now().UnixNano())
	return fmt.Sprintf("%06d", rand.Intn(1000000))
}

// SetResetCode sets the reset code for a given email in Redis with an expiration time of 15 minutes.
func SetResetCode(ctx context.Context, email, code string) error {
	cacheInstance, err := cache.NewCache()
	if err != nil {
		return err
	}
	// Use the Cache's Set method
	return cacheInstance.Set(ctx, "reset_code:"+email, code, 15*time.Minute)
}

// GetResetCode retrieves the reset code for a given email from Redis.
func GetResetCode(ctx context.Context, email string) (*string, error) {
	cacheInstance, err := cache.NewCache()
	if err != nil {
		return nil, err
	}
	// Use the Cache's Get method
	code, err := cacheInstance.Get(ctx, "reset_code:"+email)
	if err != nil {
		return nil, err
	}
	if code == "" {
		return nil, nil // Return nil if the code does not exist
	}
	return &code, nil
}

// DeleteResetCode deletes the reset code for a given email from Redis.
func DeleteResetCode(ctx context.Context, email string) error {
	cacheInstance, err := cache.NewCache()
	if err != nil {
		return err
	}
	// Use the Cache's Delete method
	return cacheInstance.Delete(ctx, "reset_code:"+email)
}
