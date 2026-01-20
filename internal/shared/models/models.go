package models

import "time"

type RateLimit struct {
	WindowSeconds int `json:"window_seconds"`
	MaxRequests   int `json:"max_requests"`
}

type Policy struct {
	ID              string     `json:"id"`
	Name            string     `json:"name"`
	RoutePrefix     string     `json:"route_prefix"`
	RequiredScopes  []string   `json:"required_scopes"`
	RequiredVCTypes []string   `json:"required_vc_types,omitempty"`
	AllowedIssuers  []string   `json:"allowed_issuers,omitempty"`
	MinTrustTier    *int       `json:"min_trust_tier,omitempty"`
	RateLimit       *RateLimit `json:"rate_limit,omitempty"`
	TokenTTLSeconds int        `json:"token_ttl_seconds"`
}

type Issuer struct {
	DID       string    `json:"did"`
	PublicKey string    `json:"public_key"`
	Enabled   bool      `json:"enabled"`
	TrustTier int       `json:"trust_tier"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type RevocationList struct {
	ListID    string    `json:"listId"`
	Revoked   []string  `json:"revoked"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type ChallengeResponse struct {
	Challenge string `json:"challenge"`
	Nonce     string `json:"nonce"`
	ExpiresAt int64  `json:"expiresAt"`
	Audience  string `json:"audience"`
	Domain    string `json:"domain"`
}

type AuthVerifyRequest struct {
	DID          string   `json:"did"`
	Challenge    string   `json:"challenge"`
	Signature    string   `json:"signature"`
	Scopes       []string `json:"scopes,omitempty"`
	Credential   string   `json:"credential,omitempty"`
	Presentation string   `json:"presentation,omitempty"`
}

type AuthVerifyResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"`
}

type AccessTokenClaims struct {
	Subject     string   `json:"sub"`
	Scopes      []string `json:"scopes"`
	VCTypes     []string `json:"vc_types,omitempty"`
	VCIssuer    string   `json:"vc_issuer,omitempty"`
	VCTrustTier int      `json:"vc_trust_tier,omitempty"`
	Issuer      string   `json:"iss"`
	IssuedAt    int64    `json:"iat"`
	ExpiresAt   int64    `json:"exp"`
	JWTID       string   `json:"jti"`
	KeyID       string   `json:"kid,omitempty"` // Signing key ID (for rotation tracking)
}

type CredentialClaims struct {
	Issuer   string                 `json:"iss"`
	Subject  string                 `json:"sub"`
	IssuedAt int64                  `json:"iat"`
	Expiry   int64                  `json:"exp"`
	JWTID    string                 `json:"jti"`
	VC       map[string]interface{} `json:"vc"`
}

type AuditEvent struct {
	Time     time.Time              `json:"time"`
	Event    string                 `json:"event"`
	Subject  string                 `json:"subject,omitempty"`
	Actor    string                 `json:"actor,omitempty"`
	Outcome  string                 `json:"outcome"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}
