package validator

import "github.com/go-playground/validator/v10"

// PlayVal wraps the playground validator for use with Echo framework.
// This was originally in main.go with the comment:
// "validation needs to be moved to a separate file"
type PlayVal struct {
	validator *validator.Validate
}

func New() *PlayVal {
	return &PlayVal{
		validator: validator.New(),
	}
}

// Validate performs struct validation using the playground validator.
// This method satisfies Echo's Validator interface.
func (v *PlayVal) Validate(i any) error {
	if err := v.validator.Struct(i); err != nil {
		return err
	}
	return nil
}
