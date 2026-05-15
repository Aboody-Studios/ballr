package validator

import (
	"testing"
)

type testStruct struct {
	Name  string `validate:"required"`
	Email string `validate:"required,email"`
	Age   int    `validate:"gte=0,lte=130"`
}

func TestValidator_Validate(t *testing.T) {
	v := New()

	t.Run("valid struct", func(t *testing.T) {
		s := testStruct{Name: "Test", Email: "test@example.com", Age: 25}
		if err := v.Validate(s); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("missing required field", func(t *testing.T) {
		s := testStruct{Name: "", Email: "test@example.com", Age: 25}
		if err := v.Validate(s); err == nil {
			t.Fatal("expected validation error")
		}
	})

	t.Run("invalid email", func(t *testing.T) {
		s := testStruct{Name: "Test", Email: "not-an-email", Age: 25}
		if err := v.Validate(s); err == nil {
			t.Fatal("expected validation error")
		}
	})

	t.Run("age out of range", func(t *testing.T) {
		s := testStruct{Name: "Test", Email: "test@example.com", Age: 200}
		if err := v.Validate(s); err == nil {
			t.Fatal("expected validation error")
		}
	})
}
