package api

import (
	"auth_client/proto"
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
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
