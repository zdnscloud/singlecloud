package authorize

import (
	"testing"
	"time"

	ut "github.com/zdnscloud/cement/unittest"
)

func TestTokenCreationAndValidation(t *testing.T) {
	repo := NewTokenRepo([]byte("gogod boy"), 10*time.Second)
	token, err := repo.CreateToken("ben")
	ut.Assert(t, err == nil, "create token shouldn't failed, but get:%v", err)

	user, err := repo.ParseToken(token)
	ut.Assert(t, err == nil, "token is valid, but get:%v", err)
	ut.Equal(t, user, "ben")

	_, err = repo.ParseToken(token + "xxx")
	ut.Assert(t, err == ErrInvalidToken, "token is invalid, but get nothing")

	repo = NewTokenRepo([]byte("gogod boy"), 2*time.Second)
	token, err = repo.CreateToken("ben")
	<-time.After(time.Second)
	extend, err := repo.RenewToken(token)
	ut.Assert(t, err == nil, "renewed token should succeed, but get:%v", err)
	<-time.After(time.Second)
	_, err = repo.ParseToken(token)
	ut.Assert(t, err == ErrExpiredToken, "token is expired, but get:%v", err)
	user, err = repo.ParseToken(extend)
	ut.Assert(t, err == nil, "renewed token is valid, but get:%v", err)
	ut.Equal(t, user, "ben")
}
