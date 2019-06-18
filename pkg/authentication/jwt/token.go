package jwt

import (
	"errors"
	"time"

	"github.com/dgrijalva/jwt-go"
)

var (
	ErrInvalidToken = errors.New("token isn't valid")
	ErrExpiredToken = errors.New("token is expired")
)

const (
	UserKey   = "user"
	ExpireKey = "expireAt"
)

type TokenRepo struct {
	secret        []byte
	validDuration time.Duration
}

func NewTokenRepo(secret []byte, validDuration time.Duration) *TokenRepo {
	return &TokenRepo{
		secret:        secret,
		validDuration: validDuration,
	}
}

func (r *TokenRepo) CreateToken(user string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		UserKey:   user,
		ExpireKey: time.Now().Add(r.validDuration).Unix(),
	})

	return token.SignedString(r.secret)
}

func (r *TokenRepo) ParseToken(tokenRaw string) (string, error) {
	token, err := jwt.Parse(tokenRaw, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return "", ErrInvalidToken
		}
		return r.secret, nil
	})

	if err != nil || token.Valid == false {
		return "", ErrInvalidToken
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if ok == false {
		return "", ErrInvalidToken
	}

	if expire_, ok := claims[ExpireKey]; ok == false {
		return "", ErrInvalidToken
	} else if expire, ok := expire_.(float64); ok == false {
		return "", ErrInvalidToken
	} else {
		expireTime := time.Unix(int64(expire), 0)
		if time.Now().After(expireTime) {
			return "", ErrExpiredToken
		}
	}

	if user_, ok := claims[UserKey]; ok == false {
		return "", ErrInvalidToken
	} else if user, ok := user_.(string); ok == false || user == "" {
		return "", ErrInvalidToken
	} else {
		return user, nil
	}
}

func (r *TokenRepo) RenewToken(tokenRaw string) (string, error) {
	if user, err := r.ParseToken(tokenRaw); err != nil {
		return "", err
	} else {
		return r.CreateToken(user)
	}
}
