package handler

import (
	"github.com/sudobytemebaby/efir/services/auth/internal/service"
	authv1 "github.com/sudobytemebaby/efir/services/shared/gen/auth"
)

func registerResponseToProto(acc *service.Account, tokens *service.TokenPair) *authv1.RegisterResponse {
	return &authv1.RegisterResponse{
		UserId:       acc.ID.String(),
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
	}
}

func loginResponseToProto(acc *service.Account, tokens *service.TokenPair) *authv1.LoginResponse {
	return &authv1.LoginResponse{
		UserId:       acc.ID.String(),
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
	}
}
