package ports

import "context"

// ReadinessChecker проверяет готовность identity-сервиса принимать трафик.
type ReadinessChecker interface {
	Check(ctx context.Context) error
}
