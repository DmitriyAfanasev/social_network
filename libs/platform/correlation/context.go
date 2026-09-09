// Package correlation содержит переносимый между слоями correlation ID контекста.
package correlation

import "context"

type contextKey struct{}

// WithID добавляет correlation ID в контекст операции.
func WithID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, contextKey{}, id)
}

// ID возвращает correlation ID операции или пустую строку.
func ID(ctx context.Context) string {
	id, ok := ctx.Value(contextKey{}).(string)
	if !ok {
		return ""
	}
	return id
}
