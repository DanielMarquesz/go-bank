package main

import (
	"math/rand"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type LoginRequest struct {
	Number   int    `json:"number"`
	Password string `json:"password"`
}

type TransferRequest struct {
	ToAccount int `json:"to_account"`
	Amount    int `json:"amount"`
}

type CreateAccountRequest struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Password  string `json:"password"`
}

type Account struct {
	ID                int       `json:"id"`
	FirstName         string    `json:"first_name"`
	LastName          string    `json:"last_name"`
	Number            int64     `json:"number"`
	Balance           int64     `json:"balance"`
	EncryptedPassword string    `json:"-"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

func NewAccount(firstName, lastName, password string) (*Account, error) {
	encpw, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	return &Account{
		FirstName:         firstName,
		LastName:          lastName,
		EncryptedPassword: string(encpw),
		Number:            int64(rand.Intn(10000)),
		CreatedAt:         time.Now().UTC(),
		UpdatedAt:         time.Now().UTC(),
	}, nil
}
