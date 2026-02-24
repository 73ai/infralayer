package devicesvc

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/73ai/infralayer/services/backend/internal/devicesvc/domain"
	"github.com/google/uuid"
)

const (
	DeviceCodeExpiry   = 15 * time.Minute
	AccessTokenExpiry  = 7 * 24 * time.Hour  // 7 days
	RefreshTokenExpiry = 30 * 24 * time.Hour // 30 days
	DeviceCodeLength   = 32
	UserCodeLength     = 8
	AccessTokenLength  = 32
	RefreshTokenLength = 32
)

type Service struct {
	deviceCodeRepo  domain.DeviceCodeRepository
	deviceTokenRepo domain.DeviceTokenRepository
}

func NewService(
	deviceCodeRepo domain.DeviceCodeRepository,
	deviceTokenRepo domain.DeviceTokenRepository,
) *Service {
	return &Service{
		deviceCodeRepo:  deviceCodeRepo,
		deviceTokenRepo: deviceTokenRepo,
	}
}

type InitiateDeviceFlowResult struct {
	DeviceCode      string
	UserCode        string
	VerificationURL string
	ExpiresIn       int
	Interval        int
}

func (s *Service) InitiateDeviceFlow(ctx context.Context) (InitiateDeviceFlowResult, error) {
	deviceCode, err := generateSecureToken(DeviceCodeLength)
	if err != nil {
		return InitiateDeviceFlowResult{}, fmt.Errorf("failed to generate device code: %w", err)
	}

	userCode, err := generateUserCode()
	if err != nil {
		return InitiateDeviceFlowResult{}, fmt.Errorf("failed to generate user code: %w", err)
	}

	now := time.Now()
	code := domain.DeviceCode{
		ID:         uuid.New(),
		DeviceCode: deviceCode,
		UserCode:   userCode,
		Status:     domain.DeviceCodeStatusPending,
		ExpiresAt:  now.Add(DeviceCodeExpiry),
		CreatedAt:  now,
	}

	if err := s.deviceCodeRepo.Create(ctx, code); err != nil {
		return InitiateDeviceFlowResult{}, fmt.Errorf("failed to create device code: %w", err)
	}

	return InitiateDeviceFlowResult{
		DeviceCode:      deviceCode,
		UserCode:        userCode,
		VerificationURL: "/cli/verify",
		ExpiresIn:       int(DeviceCodeExpiry.Seconds()),
		Interval:        5,
	}, nil
}

type PollDeviceFlowResult struct {
	Authorized     bool
	AccessToken    string
	RefreshToken   string
	ExpiresIn      int
	OrganizationID uuid.UUID
	UserID         uuid.UUID
}

func (s *Service) PollDeviceFlow(ctx context.Context, deviceCode string) (PollDeviceFlowResult, error) {
	code, err := s.deviceCodeRepo.GetByDeviceCode(ctx, deviceCode)
	if err != nil {
		return PollDeviceFlowResult{}, err
	}

	if code.Status == domain.DeviceCodeStatusExpired || code.ExpiresAt.Before(time.Now()) {
		return PollDeviceFlowResult{}, domain.ErrDeviceCodeExpired
	}

	if code.Status == domain.DeviceCodeStatusUsed {
		return PollDeviceFlowResult{}, domain.ErrDeviceCodeUsed
	}

	if code.Status == domain.DeviceCodeStatusPending {
		return PollDeviceFlowResult{}, domain.ErrAuthorizationPending
	}

	if code.Status != domain.DeviceCodeStatusAuthorized {
		return PollDeviceFlowResult{}, fmt.Errorf("unexpected device code status: %s", code.Status)
	}

	accessToken, err := generateSecureToken(AccessTokenLength)
	if err != nil {
		return PollDeviceFlowResult{}, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, err := generateSecureToken(RefreshTokenLength)
	if err != nil {
		return PollDeviceFlowResult{}, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	now := time.Now()
	token := domain.DeviceToken{
		ID:             uuid.New(),
		AccessToken:    accessToken,
		RefreshToken:   refreshToken,
		OrganizationID: code.OrganizationID,
		UserID:         code.UserID,
		ExpiresAt:      now.Add(AccessTokenExpiry),
		CreatedAt:      now,
	}

	if err := s.deviceTokenRepo.Create(ctx, token); err != nil {
		return PollDeviceFlowResult{}, fmt.Errorf("failed to create device token: %w", err)
	}

	if err := s.deviceCodeRepo.MarkAsUsed(ctx, deviceCode); err != nil {
		return PollDeviceFlowResult{}, fmt.Errorf("failed to mark device code as used: %w", err)
	}

	return PollDeviceFlowResult{
		Authorized:     true,
		AccessToken:    accessToken,
		RefreshToken:   refreshToken,
		ExpiresIn:      int(AccessTokenExpiry.Seconds()),
		OrganizationID: code.OrganizationID,
		UserID:         code.UserID,
	}, nil
}

func (s *Service) AuthorizeDevice(ctx context.Context, userCode string, organizationID, userID uuid.UUID) error {
	code, err := s.deviceCodeRepo.GetByUserCode(ctx, userCode)
	if err != nil {
		return err
	}

	if code.Status != domain.DeviceCodeStatusPending {
		if code.Status == domain.DeviceCodeStatusExpired || code.ExpiresAt.Before(time.Now()) {
			return domain.ErrDeviceCodeExpired
		}
		return domain.ErrDeviceCodeUsed
	}

	return s.deviceCodeRepo.Authorize(ctx, userCode, organizationID, userID)
}

type ValidateTokenResult struct {
	OrganizationID uuid.UUID
	UserID         uuid.UUID
}

func (s *Service) ValidateToken(ctx context.Context, accessToken string) (ValidateTokenResult, error) {
	token, err := s.deviceTokenRepo.GetByAccessToken(ctx, accessToken)
	if err != nil {
		return ValidateTokenResult{}, err
	}

	if token.RevokedAt != nil {
		return ValidateTokenResult{}, domain.ErrDeviceTokenRevoked
	}

	if token.ExpiresAt.Before(time.Now()) {
		return ValidateTokenResult{}, domain.ErrDeviceTokenExpired
	}

	return ValidateTokenResult{
		OrganizationID: token.OrganizationID,
		UserID:         token.UserID,
	}, nil
}

type RefreshTokenResult struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int
}

func (s *Service) RefreshToken(ctx context.Context, refreshToken string) (RefreshTokenResult, error) {
	token, err := s.deviceTokenRepo.GetByRefreshToken(ctx, refreshToken)
	if err != nil {
		return RefreshTokenResult{}, err
	}

	if token.RevokedAt != nil {
		return RefreshTokenResult{}, domain.ErrDeviceTokenRevoked
	}

	newAccessToken, err := generateSecureToken(AccessTokenLength)
	if err != nil {
		return RefreshTokenResult{}, fmt.Errorf("failed to generate access token: %w", err)
	}

	newRefreshToken, err := generateSecureToken(RefreshTokenLength)
	if err != nil {
		return RefreshTokenResult{}, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	newToken := domain.DeviceToken{
		AccessToken:  newAccessToken,
		RefreshToken: newRefreshToken,
		ExpiresAt:    time.Now().Add(AccessTokenExpiry),
	}

	if err := s.deviceTokenRepo.UpdateTokens(ctx, refreshToken, newToken); err != nil {
		return RefreshTokenResult{}, fmt.Errorf("failed to update tokens: %w", err)
	}

	return RefreshTokenResult{
		AccessToken:  newAccessToken,
		RefreshToken: newRefreshToken,
		ExpiresIn:    int(AccessTokenExpiry.Seconds()),
	}, nil
}

func (s *Service) RevokeToken(ctx context.Context, accessToken string) error {
	return s.deviceTokenRepo.Revoke(ctx, accessToken)
}

func generateSecureToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(bytes), nil
}

func generateUserCode() (string, error) {
	const charset = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	code := make([]byte, UserCodeLength)
	for i := range code {
		b := make([]byte, 1)
		if _, err := rand.Read(b); err != nil {
			return "", err
		}
		code[i] = charset[int(b[0])%len(charset)]
	}
	formatted := string(code[:4]) + "-" + string(code[4:])
	return strings.ToUpper(formatted), nil
}
