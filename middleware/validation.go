package middleware

import (
	"fmt"
	"net/http"

	"github.com/pocketbase/pocketbase/core"
)

// ValidatePhraseLength middleware ensures the passphrase meets minimum length requirements
func ValidatePhraseLength(minLength int) func(next func(e *core.RequestEvent) error) func(e *core.RequestEvent) error {
	return func(next func(e *core.RequestEvent) error) func(e *core.RequestEvent) error {
		return func(e *core.RequestEvent) error {
			// Extract the phrase from the path
			phrase := e.Request.PathValue("phrase")

			// Validate the phrase length
			if len(phrase) < minLength {
				return e.JSON(http.StatusBadRequest, map[string]string{
					"error": fmt.Sprintf("Passphrase must be at least %d characters long", minLength),
				})
			}

			// Continue with the next handler
			return next(e)
		}
	}
}
