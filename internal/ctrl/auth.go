package ctrl

import (
	"context"
	"errors"
	"time"

	"github.com/JMURv/golang-clean-template/internal/auth"
	"github.com/JMURv/golang-clean-template/internal/dto"
	md "github.com/JMURv/golang-clean-template/internal/models"
	"github.com/JMURv/golang-clean-template/internal/repo"
	"github.com/google/uuid"
	"github.com/opentracing/opentracing-go"
	"go.uber.org/zap"
)

type authCtrl interface {
	GenPair(ctx context.Context, d *dto.DeviceRequest, uid uuid.UUID) (dto.TokenPair, error)
	Authenticate(
		ctx context.Context,
		d *dto.DeviceRequest,
		req *dto.EmailAndPasswordRequest,
	) (*dto.TokenPair, error)
	Refresh(
		ctx context.Context,
		d *dto.DeviceRequest,
		req *dto.RefreshRequest,
	) (*dto.TokenPair, error)
	Logout(ctx context.Context, uid uuid.UUID) error
}

type authRepo interface {
	CreateToken(
		ctx context.Context,
		userID uuid.UUID,
		hashedT string,
		expiresAt time.Time,
		device *md.Device,
	) error
	IsTokenValid(ctx context.Context, userID uuid.UUID, d *md.Device, token string) (bool, error)
	RevokeAllTokens(ctx context.Context, userID uuid.UUID) error
}

func (c *Controller) GenPair(
	ctx context.Context,
	d *dto.DeviceRequest,
	uid uuid.UUID,
) (dto.TokenPair, error) {
	const op = "auth.GenPair.ctrl"

	span, ctx := opentracing.StartSpanFromContext(ctx, op)
	defer span.Finish()

	var res dto.TokenPair
	access, refresh, err := c.au.GenPair(ctx, uid)
	if err != nil {
		return res, err
	}

	device := auth.GenerateDevice(d)

	err = c.repo.CreateToken(ctx, uid, refresh, c.au.GetRefreshTime(), &device)
	if err != nil {
		return res, err
	}

	res.Access = access
	res.Refresh = refresh

	return res, nil
}

func (c *Controller) Authenticate(
	ctx context.Context,
	d *dto.DeviceRequest,
	req *dto.EmailAndPasswordRequest,
) (*dto.TokenPair, error) {
	const op = "auth.Authenticate.ctrl"

	span, ctx := opentracing.StartSpanFromContext(ctx, op)
	defer span.Finish()

	res, err := c.repo.GetUserByEmail(ctx, req.Email)
	if err != nil && errors.Is(err, repo.ErrNotFound) {
		return nil, ErrNotFound
	} else if err != nil {
		return nil, err
	}

	err = c.au.ComparePasswords([]byte(res.Password), []byte(req.Password))
	if err != nil {
		return nil, auth.ErrInvalidCredentials
	}

	pair, err := c.GenPair(ctx, d, res.ID)
	if err != nil {
		return nil, err
	}

	return &dto.TokenPair{
		Access:  pair.Access,
		Refresh: pair.Refresh,
	}, nil
}

func (c *Controller) Refresh(
	ctx context.Context,
	d *dto.DeviceRequest,
	req *dto.RefreshRequest,
) (*dto.TokenPair, error) {
	const op = "auth.Refresh.ctrl"

	span, ctx := opentracing.StartSpanFromContext(ctx, op)
	defer span.Finish()

	claims, err := c.au.ParseClaims(ctx, req.Refresh)
	if err != nil {
		return nil, err
	}

	device := auth.GenerateDevice(d)

	isValid, err := c.repo.IsTokenValid(ctx, claims.UID, &device, req.Refresh)
	if err != nil {
		return nil, err
	}

	if !isValid {
		zap.L().Info(
			"token is invalid",
			zap.String("op", op),
			zap.String("userID", claims.UID.String()),
		)

		return nil, auth.ErrTokenRevoked
	}

	access, refresh, err := c.au.GenPair(ctx, claims.UID)
	if err != nil {
		return nil, err
	}

	err = c.repo.RevokeAllTokens(ctx, claims.UID)
	if err != nil {
		return nil, err
	}

	err = c.repo.CreateToken(ctx, claims.UID, refresh, c.au.GetRefreshTime(), &device)
	if err != nil {
		return nil, err
	}

	return &dto.TokenPair{
		Access:  access,
		Refresh: refresh,
	}, nil
}

func (c *Controller) Logout(ctx context.Context, uid uuid.UUID) error {
	const op = "auth.Logout.ctrl"

	span, ctx := opentracing.StartSpanFromContext(ctx, op)
	defer span.Finish()

	err := c.repo.RevokeAllTokens(ctx, uid)
	if err != nil {
		return err
	}

	return nil
}
