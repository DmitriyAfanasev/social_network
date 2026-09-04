package ports

import "context"

// ReadinessChecker проверяет готовность profiles-сервиса принимать трафик.
type ReadinessChecker interface {
	Check(ctx context.Context) error
}
