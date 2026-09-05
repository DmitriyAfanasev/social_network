// Package postgres содержит PostgreSQL-адаптеры identity-сервиса.
package postgres

import (
	"context"
	"errors"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"general-project/identity/internal/domain"
	"general-project/identity/internal/ports"
)

// UserRepository реализует чтение пользователей через pgx.
type UserRepository struct {
	pool *pgxpool.Pool
}

// NewUserRepository создаёт PostgreSQL-адаптер пользователей.
func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

// FindByID загружает доменного пользователя по UUID.
func (r *UserRepository) FindByID(ctx context.Context, id uuid.UUID) (domain.User, error) {
	const query = `
		SELECT id, email, status, last_seen_at, created_at, updated_at
		FROM identity.users
		WHERE id = $1`

	var user domain.User
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.Email,
		&user.Status,
		&user.LastSeenAt,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.User{}, ports.ErrNotFound
	}
	if err != nil {
		return domain.User{}, err
	}
	return user, nil
}

// FindByEmail загружает пользователя и хеш пароля для сценария входа.
func (r *UserRepository) FindByEmail(ctx context.Context, email string) (domain.AuthUser, error) {
	const query = `
		SELECT u.id, u.email, u.status, u.last_seen_at, u.created_at, u.updated_at, c.password_hash
		FROM identity.users AS u
		JOIN identity.credentials AS c ON c.user_id = u.id
		WHERE u.email = $1`

	var authUser domain.AuthUser
	err := r.pool.QueryRow(ctx, query, email).Scan(
		&authUser.User.ID,
		&authUser.User.Email,
		&authUser.User.Status,
		&authUser.User.LastSeenAt,
		&authUser.User.CreatedAt,
		&authUser.User.UpdatedAt,
		&authUser.PasswordHash,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.AuthUser{}, ports.ErrNotFound
	}
	if err != nil {
		return domain.AuthUser{}, err
	}
	return authUser, nil
}

// Create сохраняет пользователя и его credentials одной транзакцией.
func (r *UserRepository) Create(ctx context.Context, user domain.User, passwordHash string, event *ports.OutboxEvent) (domain.User, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.User{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	const userQuery = `
		INSERT INTO identity.users (id, email, status)
		VALUES ($1, $2, $3)
		RETURNING created_at, updated_at`
	if err := tx.QueryRow(ctx, userQuery, user.ID, user.Email, user.Status).Scan(&user.CreatedAt, &user.UpdatedAt); err != nil {
		if isUniqueViolation(err) {
			return domain.User{}, ports.ErrAlreadyExists
		}
		return domain.User{}, err
	}

	const credentialsQuery = `INSERT INTO identity.credentials (user_id, password_hash) VALUES ($1, $2)`
	if _, err := tx.Exec(ctx, credentialsQuery, user.ID, passwordHash); err != nil {
		return domain.User{}, err
	}
	const roleQuery = `
		INSERT INTO identity.user_roles (user_id, role_id)
		SELECT $1, id FROM identity.roles WHERE name = 'user'`
	if _, err := tx.Exec(ctx, roleQuery, user.ID); err != nil {
		return domain.User{}, err
	}
	if event != nil {
		const outboxQuery = `
			INSERT INTO identity.outbox_events
				(id, event_type, aggregate_id, payload, correlation_id, available_at, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7)`
		if _, err := tx.Exec(ctx, outboxQuery, event.ID, event.EventType, event.AggregateID, event.Payload, event.CorrelationID, event.AvailableAt, event.CreatedAt); err != nil {
			return domain.User{}, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.User{}, err
	}
	return user, nil
}

// Activate подтверждает email пользователя и делает его доступным для входа.
func (r *UserRepository) Activate(ctx context.Context, id uuid.UUID) (domain.User, error) {
	const query = `
		UPDATE identity.users
		SET status = $2, updated_at = now()
		WHERE id = $1 AND status = $3
		RETURNING id, email, status, last_seen_at, created_at, updated_at`
	var user domain.User
	err := r.pool.QueryRow(ctx, query, id, domain.UserStatusActive, domain.UserStatusPending).Scan(
		&user.ID, &user.Email, &user.Status, &user.LastSeenAt, &user.CreatedAt, &user.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.User{}, ports.ErrNotFound
	}
	if err != nil {
		return domain.User{}, err
	}
	return user, nil
}

// UpdatePassword заменяет хеш пароля пользователя.
func (r *UserRepository) UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash string) error {
	const query = `
		UPDATE identity.credentials
		SET password_hash = $2, updated_at = now()
		WHERE user_id = $1`
	result, err := r.pool.Exec(ctx, query, id, passwordHash)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ports.ErrNotFound
	}
	return nil
}

// ListPermissions возвращает эффективные permissions пользователя через его роли.
func (r *UserRepository) ListPermissions(ctx context.Context, userID uuid.UUID) ([]string, error) {
	const query = `
		SELECT DISTINCT p.code
		FROM identity.permissions AS p
		JOIN identity.role_permissions AS rp ON rp.permission_id = p.id
		JOIN identity.user_roles AS ur ON ur.role_id = rp.role_id
		WHERE ur.user_id = $1
		ORDER BY p.code`
	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	permissions := make([]string, 0)
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			return nil, err
		}
		permissions = append(permissions, code)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return permissions, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
