package api

import (
	"auth_client/proto"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"unicode/utf8"
)

type authInput struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type respTokens struct {
	AccessToken   string `json:"access_token"`
	RefreshToken  string `json:"refresh_token"`
	AccessExpires int64  `json:"access_expires"`
}

type userData struct {
	ID    uint   `json:"id"`
	Login string `json:"login"`
	Admin bool   `json:"admin"`
}

func (a *Api) register(w http.ResponseWriter, r *http.Request) {
	var input authInput

	defer r.Body.Close()
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := json.Unmarshal(body, &input); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := input.validate(); err != nil {
		http.Error(w, fmt.Sprintf("validation err: %v", err), http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	tokens, err := a.AuthGRPC.Register(ctx, &proto.ReqUserData{
		Login:    input.Login,
		Password: input.Password,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respTokens := respTokens{
		AccessToken:   tokens.AccessToken,
		RefreshToken:  tokens.RefreshToken,
		AccessExpires: tokens.AccessExpires,
	}
	json.NewEncoder(w).Encode(respTokens)
}

func (a *Api) login(w http.ResponseWriter, r *http.Request) {
	var input authInput

	defer r.Body.Close()
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := json.Unmarshal(body, &input); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := input.validate(); err != nil {
		http.Error(w, fmt.Sprintf("validation err: %v", err), http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	tokens, err := a.AuthGRPC.Login(ctx, &proto.ReqUserData{
		Login:    input.Login,
		Password: input.Password,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respTokens := respTokens{
		AccessToken:   tokens.AccessToken,
		RefreshToken:  tokens.RefreshToken,
		AccessExpires: tokens.AccessExpires,
	}
	json.NewEncoder(w).Encode(respTokens)
}

func (a *Api) info(w http.ResponseWriter, r *http.Request) {
	token, err := tokenFromHeader(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	ctx := context.Background()
	data, err := a.AuthGRPC.Info(ctx, &proto.AccessToken{
		AccessToken: token,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	userData := userData{
		ID:    uint(data.Id),
		Login: data.Login,
		Admin: data.Admin,
	}
	json.NewEncoder(w).Encode(userData)
}

func (a *Api) refreshTokens(w http.ResponseWriter, r *http.Request) {
	token, err := tokenFromHeader(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	ctx := context.Background()
	tokens, err := a.AuthGRPC.RefreshTokens(ctx, &proto.RefreshToken{
		RefreshToken: token,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respTokens := respTokens{
		AccessToken:   tokens.AccessToken,
		RefreshToken:  tokens.RefreshToken,
		AccessExpires: tokens.AccessExpires,
	}
	json.NewEncoder(w).Encode(respTokens)
}

func tokenFromHeader(r *http.Request) (string, error) {
	bearer := r.Header.Get("Authorization")
	if len(bearer) > 7 && strings.ToUpper(bearer[0:6]) == "BEARER" {
		return bearer[7:], nil
	}
	return "", fmt.Errorf("jwt token not found or wrong structure")
}

func (a authInput) validate() error {
	if a.Login == "" {
		return fmt.Errorf("login is required")
	}
	if utf8.RuneCountInString(a.Login) < 5 || utf8.RuneCountInString(a.Login) > 40 {
		return fmt.Errorf("login must be from 5 to 40 characters")
	}

	if a.Password == "" {
		return fmt.Errorf("password is required")
	}
	if utf8.RuneCountInString(a.Password) < 8 || utf8.RuneCountInString(a.Login) > 40 {
		return fmt.Errorf("password must be from 8 to 40 characters")
	}

	return nil
}
