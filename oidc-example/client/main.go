package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

const (
	issuer      = "http://localhost:8080"
	clientID    = "my-client"
	clientSecret = "my-client-secret"
	redirectURI = "http://localhost:8081/callback"
)

var (
	provider *oidc.Provider
	verifier *oidc.IDTokenVerifier
	config   oauth2.Config
)

func main() {
	ctx := context.Background()

	var err error
	// Wait for the OIDC server to be ready.
	for {
		provider, err = oidc.NewProvider(ctx, issuer)
		if err == nil {
			break
		}
		log.Printf("could not connect to OIDC provider, retrying: %v", err)
		time.Sleep(1 * time.Second)
	}

	verifier = provider.Verifier(&oidc.Config{ClientID: clientID})

	config = oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Endpoint:     provider.Endpoint(),
		RedirectURL:  redirectURI,
		Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
	}

	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/login", loginHandler)
	http.HandleFunc("/callback", callbackHandler)

	log.Println("OIDC Client started on http://localhost:8081")
	log.Fatal(http.ListenAndServe(":8081", nil))
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, `<h1>OIDC Client</h1><a href="/login">Login</a>`)
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	// Generate a random state parameter.
	state := "random-state-string" // In a real app, generate a random, non-guessable string.
	http.Redirect(w, r, config.AuthCodeURL(state), http.StatusFound)
}

func callbackHandler(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	// Verify the state parameter.
	if r.URL.Query().Get("state") != "random-state-string" {
		http.Error(w, "invalid state parameter", http.StatusBadRequest)
		return
	}

	oauth2Token, err := config.Exchange(ctx, r.URL.Query().Get("code"))
	if err != nil {
		http.Error(w, "failed to exchange token: "+err.Error(), http.StatusInternalServerError)
		return
	}

	rawIDToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok {
		http.Error(w, "no id_token field in oauth2 token", http.StatusInternalServerError)
		return
	}

	idToken, err := verifier.Verify(ctx, rawIDToken)
	if err != nil {
		http.Error(w, "failed to verify id token: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var claims json.RawMessage
	if err := idToken.Claims(&claims); err != nil {
		http.Error(w, "failed to get claims: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(claims)
}
