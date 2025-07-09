package db

import (
	"context"
	"database/sql"
	"errors"
	"time"

	md "github.com/JMURv/golang-clean-template/internal/models"
	"github.com/google/uuid"
	"github.com/opentracing/opentracing-go"
	"go.uber.org/zap"
)

func (r *Repository) CreateToken(
	ctx context.Context,
	userID uuid.UUID,
	hashedT string,
	expiresAt time.Time,
	device *md.Device,
) error {
	const op = "auth.CreateToken.repo"

	span, ctx := opentracing.StartSpanFromContext(ctx, op)
	defer span.Finish()

	tx, err := r.conn.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		zap.L().Error(
			"failed to begin transaction",
			zap.String("op", op),
			zap.Error(err),
		)

		return err
	}

	defer func() {
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			zap.L().Error(
				"error while transaction rollback",
				zap.String("op", op),
				zap.Error(err),
			)
		}
	}()

	_, err = tx.ExecContext(
		ctx,
		createUserDevice,
		device.ID,
		userID,
		device.Name,
		device.DeviceType,
		device.OS,
		device.Browser,
		device.UA,
		device.IP,
	)
	if err != nil {
		zap.L().Error(
			"failed to create user device",
			zap.String("op", op),
			zap.Error(err),
		)

		return err
	}

	_, err = tx.ExecContext(
		ctx, createRefreshToken,
		userID, hashedT, expiresAt, device.ID,
	)
	if err != nil {
		zap.L().Error(
			"failed to create refresh token",
			zap.String("op", op),
			zap.Error(err),
		)

		return err
	}

	if err = tx.Commit(); err != nil {
		zap.L().Error(
			"failed to commit transaction",
			zap.String("op", op),
			zap.Error(err),
		)

		return err
	}

	return nil
}

func (r *Repository) IsTokenValid(
	ctx context.Context,
	userID uuid.UUID,
	d *md.Device,
	token string,
) (bool, error) {
	const op = "auth.IsTokenValid.repo"

	span, ctx := opentracing.StartSpanFromContext(ctx, op)
	defer span.Finish()

	var stored string

	err := r.conn.QueryRowContext(ctx, isValidToken, userID, d.ID).Scan(&stored)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			zap.L().Debug(
				"no token found",
				zap.String("op", op),
				zap.String("userID", userID.String()),
				zap.Any("device", d),
			)

			return false, nil
		}

		zap.L().Error(
			"failed to validate token",
			zap.String("op", op),
			zap.String("userID", userID.String()),
			zap.Any("device", d),
			zap.Error(err),
		)

		return false, err
	}

	return token == stored, nil
}

func (r *Repository) RevokeAllTokens(ctx context.Context, userID uuid.UUID) error {
	const op = "auth.RevokeAllTokens.repo"

	span, ctx := opentracing.StartSpanFromContext(ctx, op)
	defer span.Finish()

	_, err := r.conn.ExecContext(ctx, revokeToken, userID)
	if err != nil {
		zap.L().Error(
			"failed to revoke tokens",
			zap.String("op", op),
			zap.Any("userID", userID),
			zap.Error(err),
		)

		return err
	}

	return nil
}

func (r *Repository) GetByDevice(
	ctx context.Context,
	userID uuid.UUID,
	deviceID string,
) (*md.RefreshToken, error) {
	const op = "auth.GetByDevice.repo"

	span, ctx := opentracing.StartSpanFromContext(ctx, op)
	defer span.Finish()

	var token md.RefreshToken

	err := r.conn.GetContext(ctx, &token, getTokenByDevice, userID, deviceID)
	if err != nil {
		zap.L().Error(
			"failed to get token by device",
			zap.String("op", op),
			zap.String("userID", userID.String()),
			zap.String("deviceID", deviceID),
			zap.Error(err),
		)

		return nil, err
	}

	return &token, err
}

func (r *Repository) RevokeByDevice(ctx context.Context, userID uuid.UUID, deviceID string) error {
	const op = "auth.GetByDevice.repo"

	span, ctx := opentracing.StartSpanFromContext(ctx, op)
	defer span.Finish()

	_, err := r.conn.ExecContext(ctx, revokeTokenByDevice, userID, deviceID)
	if err != nil {
		zap.L().Error(
			"failed to revoke token by device",
			zap.String("op", op),
			zap.String("userID", userID.String()),
			zap.String("deviceID", deviceID),
			zap.Error(err),
		)
	}

	return err
}
