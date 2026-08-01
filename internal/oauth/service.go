package oauth

import (
	"context"
	"fmt"
	"strings"
	"unicode"

	"github.com/Teixeiraass/ground_guard_be/util"
	"github.com/coreos/go-oidc/v3/oidc"
)

const (
	ProviderGoogle = "google"
	ProviderApple  = "apple"

	appleIssuer  = "https://appleid.apple.com"
	googleIssuer = "https://accounts.google.com"
)

type Identity struct {
	Provider      string
	Subject       string
	Email         string
	FullName      string
	Username      string
	EmailVerified bool
	Picture       string
}

type Service struct {
	config util.Config
}

func NewService(config util.Config) *Service {
	return &Service{config: config}
}

func (s *Service) Authenticate(ctx context.Context, provider, idToken string) (Identity, error) {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case ProviderGoogle:
		return s.authenticateGoogle(ctx, idToken)
	case ProviderApple:
		return s.authenticateApple(ctx, idToken)
	default:
		return Identity{}, fmt.Errorf("unsupported oauth provider: %s", provider)
	}
}

func (s *Service) authenticateGoogle(ctx context.Context, idToken string) (Identity, error) {
	if s.config.GoogleClientID == "" {
		return Identity{}, fmt.Errorf("google oauth is not configured")
	}

	return s.verifyIDToken(ctx, ProviderGoogle, googleIssuer, s.config.GoogleClientID, idToken)
}

func (s *Service) authenticateApple(ctx context.Context, idToken string) (Identity, error) {
	if s.config.AppleClientID == "" {
		return Identity{}, fmt.Errorf("apple oauth is not configured")
	}

	return s.verifyIDToken(ctx, ProviderApple, appleIssuer, s.config.AppleClientID, idToken)
}

func (s *Service) verifyIDToken(ctx context.Context, providerName, issuer, clientID, rawIDToken string) (Identity, error) {
	if strings.TrimSpace(rawIDToken) == "" {
		return Identity{}, fmt.Errorf("oauth id token is required")
	}

	provider, err := oidc.NewProvider(ctx, issuer)
	if err != nil {
		return Identity{}, fmt.Errorf("oauth provider discovery failed: %w", err)
	}

	verifier := provider.Verifier(&oidc.Config{ClientID: clientID})
	idToken, err := verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return Identity{}, fmt.Errorf("oauth id token verification failed: %w", err)
	}

	var claims oauthClaims
	if err := idToken.Claims(&claims); err != nil {
		return Identity{}, fmt.Errorf("oauth claims decode failed: %w", err)
	}

	emailVerified := toBool(claims.EmailVerified)
	fullName := claims.Name
	if fullName == "" {
		fullName = fallbackFullName(claims.Email, claims.Subject)
	}

	if strings.TrimSpace(claims.Email) == "" {
		return Identity{}, fmt.Errorf("oauth token does not contain email")
	}

	return Identity{
		Provider:      providerName,
		Subject:       claims.Subject,
		Email:         claims.Email,
		FullName:      fullName,
		Username:      usernameFromEmail(claims.Email, claims.Subject),
		EmailVerified: emailVerified,
		Picture:       claims.Picture,
	}, nil
}

type oauthClaims struct {
	Subject       string `json:"sub"`
	Email         string `json:"email"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
	EmailVerified any    `json:"email_verified"`
}

func usernameFromEmail(email, subject string) string {
	localPart := strings.SplitN(strings.TrimSpace(strings.ToLower(email)), "@", 2)[0]
	if localPart == "" {
		localPart = subject
	}

	localPart = sanitizeUsername(localPart)
	if localPart == "" {
		localPart = fmt.Sprintf("oauth_%s", sanitizeUsername(subject))
	}

	return localPart
}

func fallbackFullName(email, subject string) string {
	if email != "" {
		return strings.SplitN(email, "@", 2)[0]
	}

	return fmt.Sprintf("oauth-%s", subject)
}

func sanitizeUsername(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	value = strings.Map(func(r rune) rune {
		switch {
		case unicode.IsLetter(r), unicode.IsDigit(r):
			return r
		case r == '_' || r == '-' || r == '.':
			return r
		default:
			return '_'
		}
	}, value)
	return strings.Trim(value, "_-. ")
}

func toBool(value any) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		return strings.EqualFold(typed, "true")
	case float64:
		return typed != 0
	default:
		return false
	}
}
