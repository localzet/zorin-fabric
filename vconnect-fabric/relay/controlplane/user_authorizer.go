package controlplane

import (
	"context"
	"errors"
	"strings"
)

// UserAuthorizer проверяет, имеет ли пользователь право выполнять управляющую операцию.
// Транспорт звонка остаётся отдельным слоем и не считается доказательством личности.
type UserAuthorizer interface {
	AuthorizeUser(ctx context.Context, userID, credential string) (string, error)
}

type UserAuthorizerFunc func(ctx context.Context, userID, credential string) (string, error)

func (fn UserAuthorizerFunc) AuthorizeUser(ctx context.Context, userID, credential string) (string, error) {
	if fn == nil {
		return "", errors.New("controlplane: user authorizer is not configured")
	}
	return fn(ctx, userID, credential)
}

func normalizeUserID(value string) string {
	return strings.TrimSpace(value)
}
