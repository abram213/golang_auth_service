package main

import (
	"auth/proto"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"golang.org/x/crypto/bcrypt"
	"time"
)

//todo: write comments, structure code

type user struct {
	id 				string
	login 			string
	passwordHash 	string
	admin 			bool
}

type userClaims struct {
	ID 		string 		`json:"id"`
	Admin   bool 		`json:"admin"`
	jwt.StandardClaims
}

func (u *user) refreshTokens(config jwtConfig) (*proto.Tokens, error) {
	//set access claims
	accessExp := time.Now().Add(time.Minute * time.Duration(config.accessExpMin)).Unix()
	accessClaims := userClaims{
		u.id,
		u.admin,
		jwt.StandardClaims{
			ExpiresAt: accessExp,
		},
	}
	//generate access token
	accessToken, err := generateToken([]byte(config.accessKey), accessClaims)
	if err != nil {
		return nil, fmt.Errorf("generate token error: %v", err)
	}

	//set refresh claims
	refreshExp := time.Now().Add(time.Minute * time.Duration(config.refreshExpMin)).Unix()
	refreshClaims := userClaims{
		u.id,
		u.admin,
		jwt.StandardClaims{
			ExpiresAt: refreshExp,
		},
	}
	//generate refresh token
	refreshToken, err := generateToken([]byte(config.refreshKey), refreshClaims)
	if err != nil {
		return nil, fmt.Errorf("generate token error: %v", err)
	}
	return &proto.Tokens{
		AccessToken: accessToken,
		RefreshToken: refreshToken,
		AccessExpires: accessExp,
	}, nil
}

func generateToken(key []byte, claims jwt.Claims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(key)
}

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