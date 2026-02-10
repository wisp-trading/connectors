package clob

import (
	"fmt"
	"regexp"

	"github.com/go-playground/validator/v10"
)

var (
	validate     *validator.Validate
	ethAddrRegex = regexp.MustCompile("^0x[0-9a-fA-F]{40}$")
)

func init() {
	validate = validator.New()

	// Register custom Ethereum address validator
	_ = validate.RegisterValidation("eth_addr", validateEthAddress)
}

// validateEthAddress validates an Ethereum address format
func validateEthAddress(fl validator.FieldLevel) bool {
	addr := fl.Field().String()
	return ethAddrRegex.MatchString(addr)
}

// ValidateStruct validates a struct using the validator tags
func ValidateStruct(s interface{}) error {
	if err := validate.Struct(s); err != nil {
		if validationErrors, ok := err.(validator.ValidationErrors); ok {
			return formatValidationErrors(validationErrors)
		}
		return err
	}
	return nil
}

// formatValidationErrors converts validator errors into a readable format
func formatValidationErrors(errs validator.ValidationErrors) error {
	if len(errs) == 0 {
		return nil
	}

	// Return first error with clear message
	firstErr := errs[0]
	field := firstErr.Field()
	tag := firstErr.Tag()

	switch tag {
	case "required":
		return fmt.Errorf("%s is required", field)
	case "eth_addr":
		return fmt.Errorf("%s must be a valid Ethereum address (0x followed by 40 hex characters)", field)
	case "oneof":
		return fmt.Errorf("%s must be one of: %s", field, firstErr.Param())
	case "required_without":
		return fmt.Errorf("either %s or %s is required", field, firstErr.Param())
	default:
		return fmt.Errorf("%s failed validation: %s", field, tag)
	}
}
