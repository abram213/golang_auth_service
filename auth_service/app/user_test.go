package app

import (
	"github.com/dgrijalva/jwt-go"
	"testing"
	"time"
)

func TestHashAndCheckPassword(t *testing.T) {
	pass := "user_password"
	user := User{
		PasswordHash: pass,
	}

	if err := user.HashPassword(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !user.PasswordIsValid(pass) {
		t.Errorf("user password hash invalid")
	}
}

func TestUserIDFromToken(t *testing.T) {
	key := "test_key"
	userID := "test_user_id"

	exp := time.Now().Add(time.Minute * time.Duration(5)).Unix()
	claims := UserClaims{
		userID,
		false,
		jwt.StandardClaims{
			ExpiresAt: exp,
		},
	}

	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token, err := tok.SignedString([]byte(key))
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	tUserID, err := UserIDFromToken(token, key)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if tUserID != userID {
		t.Errorf("expected user id: %s, got: %s\n", userID, tUserID)
	}
}

func TestUserIDFromTokenErrors(t *testing.T) {
	key := "test_key"

	expired := time.Now().Add(time.Minute * time.Duration(-1)).Unix()
	claims := UserClaims{
		"test",
		false,
		jwt.StandardClaims{
			ExpiresAt: expired,
		},
	}
	expiredToken, err := generateToken([]byte(key), claims)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	tokenWithNilClaims, err := generateToken([]byte(key), nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	cases := []struct {
		token  string
		errStr string
	}{
		//malformed token
		{
			"malformed_token",
			"couldn't parse token",
		},
		//expired token
		{
			expiredToken,
			"token is either expired or not active yet",
		},
		//bad token signature
		{
			"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJuYW1lIjoiSm9obiBEb2UifQ.DBJ__________fmGpEi-JiNKByy6",
			"couldn't handle this token",
		},
		//token with claims without user id
		{
			tokenWithNilClaims,
			"claims bad structure or user id is not set",
		},
	}

	for i, c := range cases {
		_, err := UserIDFromToken(c.token, key)
		if err == nil {
			t.Errorf("[%v] expected error, got: %v", i, err)
			return
		}
		if err.Error() != c.errStr {
			t.Errorf("[%v] expected error: %v, got: %v", i, c.errStr, err)
		}

	}
}
