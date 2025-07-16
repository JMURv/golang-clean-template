package ctrl

import (
	"context"
	"errors"

	"github.com/JMURv/golang-clean-template/internal/dto"
	md "github.com/JMURv/golang-clean-template/internal/models"
	"github.com/JMURv/golang-clean-template/internal/repo"
	"github.com/google/uuid"
	"github.com/opentracing/opentracing-go"
)

type deviceCtrl interface {
	ListDevices(ctx context.Context, uid uuid.UUID) ([]md.Device, error)
	GetDevice(ctx context.Context, uid uuid.UUID, dID string) (*md.Device, error)
	GetDeviceByID(ctx context.Context, dID string) (*md.Device, error)
	UpdateDevice(ctx context.Context, uid uuid.UUID, dID string, req *dto.UpdateDeviceRequest) error
	DeleteDevice(ctx context.Context, uid uuid.UUID, dID string) error
}

type deviceRepo interface {
	GetByDevice(ctx context.Context, userID uuid.UUID, deviceID string) (*md.RefreshToken, error)
	RevokeByDevice(ctx context.Context, userID uuid.UUID, deviceID string) error

	ListDevices(ctx context.Context, uid uuid.UUID) ([]md.Device, error)
	GetDevice(ctx context.Context, uid uuid.UUID, dID string) (*md.Device, error)
	GetDeviceByID(ctx context.Context, dID string) (*md.Device, error)
	UpdateDevice(ctx context.Context, uid uuid.UUID, dID string, req *dto.UpdateDeviceRequest) error
	DeleteDevice(ctx context.Context, uid uuid.UUID, deviceID string) error
}

func (c *Controller) ListDevices(ctx context.Context, uid uuid.UUID) ([]md.Device, error) {
	const op = "devices.ListDevices.ctrl"

	span, ctx := opentracing.StartSpanFromContext(ctx, op)
	defer span.Finish()

	res, err := c.repo.ListDevices(ctx, uid)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (c *Controller) GetDevice(ctx context.Context, uid uuid.UUID, dID string) (*md.Device, error) {
	const op = "devices.GetDevice.ctrl"

	span, ctx := opentracing.StartSpanFromContext(ctx, op)
	defer span.Finish()

	res, err := c.repo.GetDevice(ctx, uid, dID)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, ErrNotFound
		}

		return nil, err
	}

	return res, nil
}

func (c *Controller) GetDeviceByID(ctx context.Context, dID string) (*md.Device, error) {
	const op = "devices.GetDeviceByID.ctrl"

	span, ctx := opentracing.StartSpanFromContext(ctx, op)
	defer span.Finish()

	res, err := c.repo.GetDeviceByID(ctx, dID)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, ErrNotFound
		}

		return nil, err
	}

	return res, nil
}

func (c *Controller) UpdateDevice(
	ctx context.Context,
	uid uuid.UUID,
	dID string,
	req *dto.UpdateDeviceRequest,
) error {
	const op = "devices.UpdateDevice.ctrl"

	span, ctx := opentracing.StartSpanFromContext(ctx, op)
	defer span.Finish()

	err := c.repo.UpdateDevice(ctx, uid, dID, req)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return ErrNotFound
		}

		return err
	}

	return nil
}

func (c *Controller) DeleteDevice(ctx context.Context, uid uuid.UUID, dID string) error {
	const op = "devices.DeleteDevice.ctrl"

	span, ctx := opentracing.StartSpanFromContext(ctx, op)
	defer span.Finish()

	err := c.repo.DeleteDevice(ctx, uid, dID)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return ErrNotFound
		}

		return err
	}

	return nil
}
