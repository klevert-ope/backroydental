package utils

import (
	"RoyDental/models"
	"errors"
	"log"
	"regexp"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
)

// Validation errors
var (
	ErrPasswordTooShort   = errors.New("password must be at least 8 characters long")
	ErrPasswordNotComplex = errors.New("password must include at least one uppercase letter, one lowercase letter, one digit, and one special character")
	ErrInvalidResetCode   = errors.New("invalid reset code")
)

// ValidateUserData validates user data using ozzo-validation.
func ValidateUserData(user models.User) error {
	err := validation.ValidateStruct(&user,
		validation.Field(&user.Username, validation.Required, validation.Length(3, 50)),
		validation.Field(&user.Email, validation.Required, is.Email),
		// Ensure password is required and follows the custom validation
		validation.Field(&user.Password, validation.Required.Error("password cannot be blank"), validation.By(validatePassword)),
	)
	if err != nil {
		log.Printf("Validation error: %v\n", err)
	}
	return err
}

// ValidatePasswordReset validates the reset code and new password.
func ValidatePasswordReset(resetCode, newPassword string) error {
	err := validation.Errors{
		"resetCode": validation.Validate(resetCode, validation.Required.Error("invalid reset code")),
		"password":  validation.Validate(newPassword, validation.Required, validation.By(validatePassword)),
	}.Filter()
	if err != nil {
		log.Printf("Validation error: %v\n", err)
	}
	return err
}

// validatePassword checks the password for length and complexity.
func validatePassword(value interface{}) error {
	password, _ := value.(string)

	if len(password) < 8 {
		log.Println("Password too short")
		return ErrPasswordTooShort
	}

	// Check complexity with regex
	var (
		lowercaseRegex = regexp.MustCompile(`[a-z]`)
		uppercaseRegex = regexp.MustCompile(`[A-Z]`)
		digitRegex     = regexp.MustCompile(`\d`)
		specialRegex   = regexp.MustCompile(`[@$!%*?&]`)
	)

	if !lowercaseRegex.MatchString(password) {
		log.Println("Password missing lowercase letter")
	}
	if !uppercaseRegex.MatchString(password) {
		log.Println("Password missing uppercase letter")
	}
	if !digitRegex.MatchString(password) {
		log.Println("Password missing digit")
	}
	if !specialRegex.MatchString(password) {
		log.Println("Password missing special character")
	}

	if !lowercaseRegex.MatchString(password) ||
		!uppercaseRegex.MatchString(password) ||
		!digitRegex.MatchString(password) ||
		!specialRegex.MatchString(password) {
		return ErrPasswordNotComplex
	}

	return nil
}
