package validate

import (
	"errors"
	"fmt"
)

var errRequired = errors.New("value is required and cannot be empty")

// Names are not intended to match flags naming
// these names are used only as a hint to which value might be missing.
func requiredError(name string) error {
	return fmt.Errorf("%s: %w", name, errRequired)
}

func RequiredString(value, name string) error {
	if value == "" {
		return requiredError(name)
	}

	return nil
}
