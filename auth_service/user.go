package main

import (
	"auth/proto"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"golang.org/x/crypto/bcrypt"
	"math/rand"
	"time"
)

type user struct {
	id           string
	login        string
	passwordHash string
	admin        bool
}

type userClaims struct {
	ID    string `json:"id"`
	Admin bool   `json:"admin"`
	jwt.StandardClaims
}

//refreshTokens generate new user access and refresh tokens
func (u *user) refreshTokens(config jwtConfig) (*proto.Tokens, error) {
	aToken, aExp, err := u.genToken([]byte(config.accessKey), config.accessExpMin)
	if err != nil {
		return nil, fmt.Errorf("generate access token error: %v", err)
	}

	rToken, _, err := u.genToken([]byte(config.refreshKey), config.refreshExpMin)
	if err != nil {
		return nil, fmt.Errorf("generate refresh token error: %v", err)
	}

	return &proto.Tokens{
		AccessToken:   aToken,
		RefreshToken:  rToken,
		AccessExpires: aExp,
	}, nil
}

//hashPassword hashes user password
func (u *user) hashPassword() error {
	hashedPass, err := bcrypt.GenerateFromPassword([]byte(u.passwordHash), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.passwordHash = string(hashedPass)
	return nil
}

//passwordIsValid check user password
func (u *user) passwordIsValid(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.passwordHash), []byte(password))
	return err == nil
}

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ*#$^&@!")

//generateID generates user id
func (u *user) generateID() error {
	rand.Seed(time.Now().UnixNano())
	id := make([]rune, 30)
	for i := range id {
		id[i] = letters[rand.Intn(len(letters))]
	}
	u.id = string(id)
	return nil
}

func (u *user) genToken(key []byte, expMin int) (string, int64, error) {
	//set claims
	exp := time.Now().Add(time.Minute * time.Duration(expMin)).Unix()
	claims := userClaims{
		u.id,
		u.admin,
		jwt.StandardClaims{
			ExpiresAt: exp,
		},
	}
	//generate  token
	token, err := generateToken(key, claims)
	if err != nil {
		return "", 0, fmt.Errorf("generate token error: %v", err)
	}
	return token, exp, nil
}

func generateToken(key []byte, claims jwt.Claims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(key)
}

//userIDFromToken parse token string, validate and get user id from claims
func userIDFromToken(tokenString string, key string) (string, error) {
	token, err := jwt.ParseWithClaims(tokenString, &userClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(key), nil
	})

	if !token.Valid {
		if ve, ok := err.(*jwt.ValidationError); ok {
			if ve.Errors&jwt.ValidationErrorMalformed != 0 {
				return "", fmt.Errorf("bad token")
			} else if ve.Errors&(jwt.ValidationErrorExpired|jwt.ValidationErrorNotValidYet) != 0 {
				// Token is either expired or not active yet
				return "", fmt.Errorf("token is either expired or not active yet")
			} else {
				return "", fmt.Errorf("couldn't handle this token")
			}
		}
	}

	claims, ok := token.Claims.(*userClaims)
	if !ok {
		return "", fmt.Errorf("can`t parse token claims")
	}
	return claims.ID, nil
}
