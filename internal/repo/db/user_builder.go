package db

import (
	"context"

	sq "github.com/Masterminds/squirrel"
	"github.com/opentracing/opentracing-go"
	"go.uber.org/zap"
)

type userListQuery struct {
	countQ    string
	countArgs []any
	dataQ     string
	dataArgs  []any
}

func buildUserListQuery(
	ctx context.Context,
	page, size int,
	filters map[string]any,
) (userListQuery, error) {
	const op = "users.buildUserListQuery.repo"

	span, _ := opentracing.StartSpanFromContext(ctx, op)
	defer span.Finish()

	query := sq.Select().From("users u").PlaceholderFormat(sq.Dollar)

	if isActive, ok := filters["is_active"].(bool); ok {
		query = query.Where(sq.Eq{"u.is_active": isActive})
	}

	if isVerified, ok := filters["is_email_verified"].(bool); ok {
		query = query.Where(sq.Eq{"u.is_email_verified": isVerified})
	}

	countSql, countArgs, err := query.Columns("COUNT(DISTINCT u.id)").ToSql()
	if err != nil {
		span.SetTag("error", true)
		zap.L().Error("failed to build count query", zap.String("op", op), zap.Error(err))
		return userListQuery{}, err
	}

	dataSql, dataArgs, err := query.
		Columns(
			"u.id",
			"u.name",
			"u.email",
			"u.avatar",
			"u.is_active",
			"u.is_email_verified",
			"u.created_at",
			"u.updated_at",
		).
		Limit(uint64(size)).
		Offset(uint64((page - 1) * size)).
		ToSql()
	if err != nil {
		span.SetTag("error", true)
		zap.L().Error("failed to build data query", zap.String("op", op), zap.Error(err))
		return userListQuery{}, err
	}

	return userListQuery{
		countQ:    countSql,
		countArgs: countArgs,
		dataQ:     dataSql,
		dataArgs:  dataArgs,
	}, nil
}
