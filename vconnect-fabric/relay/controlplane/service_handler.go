package controlplane

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"vconnect/relay/tunnel"
)

type SessionController interface {
	HandleSessionCreate(ctx context.Context, userID string, request tunnel.SessionCreateRequest) (tunnel.SessionReady, error)
}

type ServiceHandler struct {
	UserID         string
	AllowedUserIDs map[string]struct{}
	Authorizer     UserAuthorizer
	CookieVault    *CookieVault
	Sessions       SessionController
}

func (h ServiceHandler) BindBridge(ctx context.Context, bridge *tunnel.RelayBridge) error {
	if bridge == nil {
		return errors.New("controlplane: relay bridge is required")
	}
	if h.Authorizer == nil && h.UserID == "" && len(h.AllowedUserIDs) == 0 {
		return errors.New("controlplane: user authorization is not configured")
	}
	if h.CookieVault != nil {
		bridge.SetOnCookieSubmit(func(submit tunnel.CookieSubmit) (tunnel.CookieAck, error) {
			userID, err := h.authorizeUser(ctx, submit.UserID, submit.Credential)
			if err != nil {
				return tunnel.CookieAck{}, err
			}
			return h.CookieVault.StoreSubmit(userID, submit)
		})
	}
	if h.Sessions != nil {
		bridge.SetOnSessionCreate(func(request tunnel.SessionCreateRequest) (tunnel.SessionReady, error) {
			userID, err := h.authorizeUser(ctx, request.UserID, request.Credential)
			if err != nil {
				return tunnel.SessionReady{}, err
			}
			return h.Sessions.HandleSessionCreate(ctx, userID, request)
		})
	}
	bridge.SetOnEgressListRequest(func(request tunnel.EgressListRequest) ([]tunnel.EgressDescriptor, error) {
		if _, err := h.authorizeUser(ctx, request.UserID, request.Credential); err != nil {
			return nil, err
		}
		return nil, nil
	})
	return nil
}

func (h ServiceHandler) authorizeUser(ctx context.Context, requested, credential string) (string, error) {
	requested = strings.TrimSpace(requested)
	if h.Authorizer != nil {
		return h.Authorizer.AuthorizeUser(ctx, requested, credential)
	}
	if len(h.AllowedUserIDs) > 0 {
		if _, ok := h.AllowedUserIDs[requested]; !ok {
			return "", fmt.Errorf("controlplane: user is not authorized")
		}
		return requested, nil
	}
	if requested != "" && requested != h.UserID {
		return "", fmt.Errorf("controlplane: user is not authorized")
	}
	return h.UserID, nil
}
