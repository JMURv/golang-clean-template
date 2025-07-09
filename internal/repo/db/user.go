package db

import (
	"context"
	"database/sql"
	"errors"

	"github.com/JMURv/golang-clean-template/internal/dto"
	md "github.com/JMURv/golang-clean-template/internal/models"
	"github.com/JMURv/golang-clean-template/internal/repo"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jmoiron/sqlx"
	"github.com/opentracing/opentracing-go"
	"go.uber.org/zap"
)

func (r *Repository) ListUsers(
	ctx context.Context,
	page, size int,
	filters map[string]any,
) (*dto.PaginatedUserResponse, error) {
	const op = "users.ListUsers.repo"

	span, ctx := opentracing.StartSpanFromContext(ctx, op)
	defer span.Finish()

	var count int64
	err := r.conn.QueryRowContext(ctx, userSelectQ).Scan(&count)
	if err != nil {
		zap.L().Error("failed to count users", zap.String("op", op), zap.Error(err))

		return nil, err
	}

	rows, err := r.conn.QueryxContext(ctx, userListQ, size, (page-1)*size)
	if err != nil {
		zap.L().Error(
			"failed to list users",
			zap.String("op", op),
			zap.Int("page", page),
			zap.Int("size", size),
			zap.Any("filters", filters),
			zap.Error(err),
		)

		return nil, err
	}
	defer func(rows *sqlx.Rows) {
		if err := rows.Close(); err != nil {
			zap.L().Error(
				"failed to close rows",
				zap.String("op", op),
				zap.Error(err),
			)
		}
	}(rows)

	res := make([]*md.User, 0, size)
	for rows.Next() {
		user := &md.User{}

		err = rows.Scan(
			&user.ID,
			&user.Name,
			&user.Email,
			&user.Avatar,
			&user.IsActive,
			&user.IsEmailVerified,
			&user.CreatedAt,
			&user.UpdatedAt,
		)
		if err != nil {
			zap.L().Error(
				"failed to scan user",
				zap.String("op", op),
				zap.Error(err),
			)

			return nil, err
		}

		res = append(res, user)
	}

	if err = rows.Err(); err != nil {
		zap.L().Error(
			"failed to scan rows",
			zap.String("op", op),
			zap.Error(err),
		)

		return nil, err
	}

	totalPages := int((count + int64(size) - 1) / int64(size))

	return &dto.PaginatedUserResponse{
		Data:        res,
		Count:       count,
		TotalPages:  totalPages,
		CurrentPage: page,
		HasNextPage: page < totalPages,
	}, nil
}

func (r *Repository) GetUserByID(ctx context.Context, userID uuid.UUID) (*md.User, error) {
	const op = "users.GetUserByID.repo"

	span, ctx := opentracing.StartSpanFromContext(ctx, op)
	defer span.Finish()

	res := &md.User{}

	err := r.conn.QueryRowContext(ctx, userGetByIDQ, userID).Scan(
		&res.ID,
		&res.Name,
		&res.Email,
		&res.Avatar,
		&res.IsActive,
		&res.IsEmailVerified,
		&res.CreatedAt,
		&res.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		zap.L().Debug(
			"no user found",
			zap.String("op", op),
			zap.String("userID", userID.String()),
		)

		return nil, repo.ErrNotFound
	} else if err != nil {
		zap.L().Error(
			"failed to get user",
			zap.String("op", op),
			zap.String("userID", userID.String()),
			zap.Error(err),
		)

		return nil, err
	}

	return res, nil
}

func (r *Repository) GetUserByEmail(ctx context.Context, email string) (*md.User, error) {
	const op = "users.GetUserByEmail.repo"

	span, ctx := opentracing.StartSpanFromContext(ctx, op)
	defer span.Finish()

	res := &md.User{}

	err := r.conn.QueryRowContext(ctx, userGetByEmailQ, email).
		Scan(
			&res.ID,
			&res.Name,
			&res.Email,
			&res.Password,
			&res.Avatar,
			&res.IsActive,
			&res.IsEmailVerified,
			&res.CreatedAt,
			&res.UpdatedAt,
		)
	if err != nil && errors.Is(err, sql.ErrNoRows) {
		zap.L().Debug(
			"no user found",
			zap.String("op", op),
			zap.String("email", email),
		)

		return nil, repo.ErrNotFound
	} else if err != nil {
		zap.L().Error(
			"failed to get user",
			zap.String("op", op),
			zap.String("email", email),
			zap.Error(err),
		)

		return nil, err
	}

	return res, nil
}

func (r *Repository) CreateUser(
	ctx context.Context,
	req *dto.CreateUserRequest,
) (uuid.UUID, error) {
	const op = "users.CreateUser.repo"

	span, ctx := opentracing.StartSpanFromContext(ctx, op)
	defer span.Finish()

	tx, err := r.conn.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		zap.L().Error(
			"failed to begin transaction",
			zap.String("op", op),
			zap.Error(err),
		)

		return uuid.Nil, err
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

	var id uuid.UUID

	err = tx.QueryRowContext(
		ctx,
		userCreateQ,
		req.Name,
		req.Password,
		req.Email,
		req.Avatar,
		req.IsActive,
		req.IsEmail,
	).Scan(&id)
	if err != nil {
		trgtErr := &pgconn.PgError{}
		if errors.As(err, &trgtErr) {
			zap.L().Debug(
				"user already exists",
				zap.String("op", op),
			)

			return uuid.Nil, repo.ErrAlreadyExists
		}

		zap.L().Error(
			"failed to create user",
			zap.String("op", op),
			zap.Error(err),
		)

		return uuid.Nil, err
	}

	if err = tx.Commit(); err != nil {
		zap.L().Error(
			"failed to commit transaction",
			zap.String("op", op),
			zap.Error(err),
		)

		return uuid.Nil, err
	}

	return id, nil
}

func (r *Repository) UpdateUser(
	ctx context.Context,
	id uuid.UUID,
	req *dto.UpdateUserRequest,
) error {
	const op = "users.UpdateUser.repo"

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
		err := tx.Rollback()
		if err != nil && !errors.Is(err, sql.ErrTxDone) ||
			!errors.Is(err, repo.ErrNotFound) {
			span.SetTag("error", true)
			zap.L().Error(
				"error while transaction rollback",
				zap.String("op", op),
				zap.Error(err),
			)
		}
	}()

	res, err := tx.ExecContext(
		ctx,
		userUpdateQ,
		req.Name,
		req.Email,
		req.Avatar,
		req.IsActive,
		req.IsEmail,
		id,
	)
	if err != nil {
		zap.L().Error(
			"failed to update user",
			zap.String("op", op),
			zap.Error(err),
		)

		return err
	}

	aff, err := res.RowsAffected()
	if err != nil {
		zap.L().Error(
			"failed to get affected rows",
			zap.String("op", op),
			zap.Error(err),
		)

		return err
	}

	if aff == 0 {
		zap.L().Debug(
			"failed to find user",
			zap.String("op", op),
		)

		return repo.ErrNotFound
	}

	err = tx.Commit()
	if err != nil {
		zap.L().Error(
			"failed to commit transaction",
			zap.String("op", op),
			zap.Error(err),
		)

		return err
	}

	return nil
}

func (r *Repository) DeleteUser(ctx context.Context, id uuid.UUID) error {
	const op = "users.DeleteUser.repo"

	span, ctx := opentracing.StartSpanFromContext(ctx, op)
	defer span.Finish()

	res, err := r.conn.ExecContext(ctx, userDeleteQ, id)
	if err != nil {
		zap.L().Error(
			"failed to delete user",
			zap.String("op", op),
			zap.Error(err),
		)

		return err
	}

	aff, err := res.RowsAffected()
	if err != nil {
		zap.L().Error(
			"failed to get affected rows",
			zap.String("op", op),
			zap.Error(err),
		)

		return err
	}

	if aff == 0 {
		zap.L().Debug(
			"failed to find user",
			zap.String("op", op),
		)

		return repo.ErrNotFound
	}

	return nil
}
