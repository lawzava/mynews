package validate

import (
	"fmt"
)

func requiredError(name string) error {
	return fmt.Errorf("'%s' is required and cannot be empty", name)
}

func RequiredString(value, name string) error {
	if value == "" {
		return requiredError(name)
	}

	return nil
}

func RequiredInt64(value int64, name string) error {
	if value == 0 {
		return requiredError(name)
	}

	return nil
}
