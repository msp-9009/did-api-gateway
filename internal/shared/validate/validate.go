package validate

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

var (
	ErrInvalidDID       = errors.New("invalid DID format")
	ErrInvalidDIDMethod = errors.New("unsupported DID method")
	ErrInvalidSignature = errors.New("invalid signature format")
	ErrInvalidScopes    = errors.New("invalid scopes")
)

// Supported DID methods
var supportedDIDMethods = map[string]bool{
	"key": true,
	"web": true,
	"ion": true,
}

// DID format: did:<method>:<method-specific-id>
var didRegex = regexp.MustCompile(`^did:([a-z0-9]+):([a-zA-Z0-9._%-]+)$`)

// Base64URL pattern (for signatures)
var base64URLRegex = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

// ValidateDID validates a DID string
func ValidateDID(did string) error {
	if did == "" {
		return ErrInvalidDID
	}

	matches := didRegex.FindStringSubmatch(did)
	if matches == nil {
		return ErrInvalidDID
	}

	method := matches[1]
	if !supportedDIDMethods[method] {
		return fmt.Errorf("%w: %s", ErrInvalidDIDMethod, method)
	}

	// Additional validation for specific methods
	switch method {
	case "key":
		// did:key uses multibase encoding (starts with 'z' for base58btc)
		methodSpecificID := matches[2]
		if !strings.HasPrefix(methodSpecificID, "z") {
			return fmt.Errorf("%w: did:key must start with 'z'", ErrInvalidDID)
		}
	case "web":
		// did:web uses domain names (optionally with path)
		methodSpecificID := matches[2]
		if len(methodSpecificID) < 3 {
			return fmt.Errorf("%w: did:web domain too short", ErrInvalidDID)
		}
	}

	return nil
}

// ValidateSignature validates a base64url-encoded signature
func ValidateSignature(signature string) error {
	if signature == "" {
		return ErrInvalidSignature
	}

	if !base64URLRegex.MatchString(signature) {
		return ErrInvalidSignature
	}

	// Ed25519 signatures are 64 bytes (86 base64url chars, or 88 with padding)
	if len(signature) < 80 || len(signature) > 100 {
		return fmt.Errorf("%w: unexpected signature length", ErrInvalidSignature)
	}

	return nil
}

// ValidateScopes validates requested scopes
func ValidateScopes(scopes []string) error {
	if len(scopes) == 0 {
		return nil // Empty scopes are allowed (will default to 'basic')
	}

	validScopes := map[string]bool{
		"basic":   true,
		"premium": true,
	}

	for _, scope := range scopes {
		if !validScopes[scope] {
			return fmt.Errorf("%w: unknown scope '%s'", ErrInvalidScopes, scope)
		}
	}

	return nil
}

// ValidateChallenge validates the challenge string format
func ValidateChallenge(challenge string) error {
	if challenge == "" {
		return errors.New("challenge cannot be empty")
	}

	// Challenge should contain required fields
	requiredFields := []string{"did=", "nonce=", "aud=", "domain=", "exp="}
	for _, field := range requiredFields {
		if !strings.Contains(challenge, field) {
			return fmt.Errorf("challenge missing required field: %s", field)
		}
	}

	return nil
}

// ValidateTTL validates a time-to-live duration
func ValidateTTL(ttl time.Duration, min, max time.Duration) error {
	if ttl < min {
		return fmt.Errorf("TTL too short: minimum is %s", min)
	}
	if ttl > max {
		return fmt.Errorf("TTL too long: maximum is %s", max)
	}
	return nil
}

// SanitizeString removes potentially dangerous characters
func SanitizeString(s string, maxLen int) string {
	// Remove null bytes and control characters
	s = strings.Map(func(r rune) rune {
		if r < 32 || r == 127 {
			return -1
		}
		return r
	}, s)

	// Limit length
	if len(s) > maxLen {
		return s[:maxLen]
	}

	return s
}
