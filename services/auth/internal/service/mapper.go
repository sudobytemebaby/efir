package service

import (
	"github.com/sudobytemebaby/efir/services/auth/internal/repository"
)

func toAccount(acc *repository.Account) *Account {
	return &Account{
		ID:        acc.ID,
		Email:     acc.Email,
		CreatedAt: acc.CreatedAt,
	}
}
