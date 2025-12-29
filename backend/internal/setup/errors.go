package setup

import "fmt"

type EnvironmentVariableMissingError struct {
	Variable string
}

func (e EnvironmentVariableMissingError) Error() string {
	return fmt.Sprintf("environment variable %q not set", e.Variable)
}

func NewEnvironmentVariableMissingError(v string) *EnvironmentVariableMissingError {
	return &EnvironmentVariableMissingError{
		Variable: v,
	}
}
