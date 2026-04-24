// Package validator provides a singleton input validation instance with
// custom rules for UUIDs, passwords, and domain-specific constraints.
package validator

import (
	"fmt"
	"regexp"
	"sync"
	"unicode"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

var (
	instance *validator.Validate
	once     sync.Once
)

// Get returns the singleton validator instance.
// Custom rules are registered exactly once.
func Get() *validator.Validate {
	once.Do(func() {
		v := validator.New()
		registerCustomRules(v)
		instance = v
	})
	return instance
}

// Validate validates a struct and returns a formatted error string or nil.
func Validate(s any) error {
	if err := Get().Struct(s); err != nil {
		return formatError(err)
	}
	return nil
}

// ── Custom validation rules ────────────────────────────────────

func registerCustomRules(v *validator.Validate) {
	// uuid: validates UUID v4 format
	v.RegisterValidation("uuid", func(fl validator.FieldLevel) bool { //nolint:errcheck
		_, err := uuid.Parse(fl.Field().String())
		return err == nil
	})

	// strong_password: min 8 chars, at least 1 upper, 1 lower, 1 digit, 1 special
	v.RegisterValidation("strong_password", func(fl validator.FieldLevel) bool { //nolint:errcheck
		return isStrongPassword(fl.Field().String())
	})

	// slug: lowercase letters, numbers, hyphens only
	slugRe := regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
	v.RegisterValidation("slug", func(fl validator.FieldLevel) bool { //nolint:errcheck
		return slugRe.MatchString(fl.Field().String())
	})
}

func isStrongPassword(password string) bool {
	if len(password) < 8 {
		return false
	}
	var (
		hasUpper   bool
		hasLower   bool
		hasDigit   bool
		hasSpecial bool
	)
	for _, r := range password {
		switch {
		case unicode.IsUpper(r):
			hasUpper = true
		case unicode.IsLower(r):
			hasLower = true
		case unicode.IsDigit(r):
			hasDigit = true
		case unicode.IsPunct(r) || unicode.IsSymbol(r):
			hasSpecial = true
		}
	}
	return hasUpper && hasLower && hasDigit && hasSpecial
}

func formatError(err error) error {
	if errs, ok := err.(validator.ValidationErrors); ok {
		for _, e := range errs {
			return fmt.Errorf("field '%s' failed validation: %s", e.Field(), e.Tag())
		}
	}
	return err
}
