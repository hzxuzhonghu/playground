package main

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"

	"gopkg.in/square/go-jose.v2"
	"gopkg.in/square/go-jose.v2/jwt"
)

var (
	// A simple in-memory store for user credentials.
	users = map[string]string{
		"user": "password",
	}

	// A simple in-memory store for authorization codes.
	authCodes = make(map[string]string)
	mu        sync.Mutex

	// The signing key for the ID tokens.
	signer jose.Signer
	jwks   jose.JSONWebKeySet
)

const (
	issuer       = "http://localhost:8080"
	clientID     = "my-client"
	clientSecret = "my-client-secret"
	redirectURI  = "http://localhost:8081/callback"
)

func main() {
	// Generate a new RSA key for signing tokens.
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		log.Fatalf("failed to generate RSA key: %v", err)
	}
	privateKey := &jose.JSONWebKey{Key: key, KeyID: "1", Algorithm: string(jose.RS256), Use: "sig"}
	publicKey := &jose.JSONWebKey{Key: &key.PublicKey, KeyID: "1", Algorithm: string(jose.RS256), Use: "sig"}
	jwks = jose.JSONWebKeySet{Keys: []jose.JSONWebKey{*publicKey}}

	// Create a new signer.
	signer, err = jose.NewSigner(jose.SigningKey{Algorithm: jose.RS256, Key: privateKey}, (&jose.SignerOptions{}).WithType("JWT"))
	if err != nil {
		log.Fatalf("failed to create signer: %v", err)
	}

	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/.well-known/openid-configuration", discoveryHandler)
	http.HandleFunc("/keys", keysHandler)
	http.HandleFunc("/auth", authHandler)
	http.HandleFunc("/token", tokenHandler)
	http.HandleFunc("/userinfo", userinfoHandler)
	http.HandleFunc("/login", loginHandler)

	log.Println("OIDC Server started on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "OIDC Server is running. Visit /login to start.")
}

func discoveryHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"issuer":                 issuer,
		"authorization_endpoint": issuer + "/auth",
		"token_endpoint":         issuer + "/token",
		"userinfo_endpoint":      issuer + "/userinfo",
		"jwks_uri":               issuer + "/keys",
		"response_types_supported": []string{
			"code",
		},
		"subject_types_supported": []string{
			"public",
		},
		"id_token_signing_alg_values_supported": []string{
			"RS256",
		},
	})
}

func keysHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(jwks)
}

func authHandler(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	if q.Get("client_id") != clientID {
		http.Error(w, "invalid client_id", http.StatusBadRequest)
		return
	}
	if q.Get("redirect_uri") != redirectURI {
		http.Error(w, "invalid redirect_uri", http.StatusBadRequest)
		return
	}
	// In a real implementation, you would check other parameters like scope, response_type, etc.

	// Redirect to login page, passing along the auth request params.
	http.Redirect(w, r, "/login?"+q.Encode(), http.StatusFound)
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, `
			<h1>Login</h1>
			<form method="post" action="/login?%s">
				<label for="username">Username:</label>
				<input type="text" id="username" name="username"><br><br>
				<label for="password">Password:</label>
				<input type="password" id="password" name="password"><br><br>
				<input type="submit" value="Login">
			</form>
		`, r.URL.RawQuery)
		return
	}

	if r.Method == http.MethodPost {
		r.ParseForm()
		username := r.FormValue("username")
		password := r.FormValue("password")

		if storedPassword, ok := users[username]; !ok || storedPassword != password {
			http.Error(w, "invalid credentials", http.StatusUnauthorized)
			return
		}

		// Generate a new authorization code.
		code := "xyz123" // In a real implementation, this should be a random, single-use code.
		mu.Lock()
		authCodes[code] = username
		mu.Unlock()

		// Redirect back to the client with the authorization code.
		q := r.URL.Query()
		redirectURL, _ := url.Parse(q.Get("redirect_uri"))
		newQuery := redirectURL.Query()
		newQuery.Set("code", code)
		newQuery.Set("state", q.Get("state"))
		redirectURL.RawQuery = newQuery.Encode()

		http.Redirect(w, r, redirectURL.String(), http.StatusFound)
	}
}

func tokenHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	code := r.FormValue("code")
	grantType := r.FormValue("grant_type")
	clientIDHeader := r.FormValue("client_id")
	clientSecretHeader := r.FormValue("client_secret")

	// Basic validation.
	if grantType != "authorization_code" {
		http.Error(w, "unsupported grant_type", http.StatusBadRequest)
		return
	}
	if clientIDHeader != clientID || clientSecretHeader != clientSecret {
		http.Error(w, "invalid client credentials", http.StatusUnauthorized)
		return
	}

	mu.Lock()
	username, ok := authCodes[code]
	if !ok {
		mu.Unlock()
		http.Error(w, "invalid authorization code", http.StatusBadRequest)
		return
	}
	delete(authCodes, code) // Code should be single-use.
	mu.Unlock()

	// Create ID token.
	claims := jwt.Claims{
		Issuer:   issuer,
		Subject:  username,
		Audience: jwt.Audience{clientID},
		Expiry:   jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
		IssuedAt: jwt.NewNumericDate(time.Now()),
	}

	idToken, err := jwt.Signed(signer).Claims(claims).CompactSerialize()
	if err != nil {
		http.Error(w, "failed to create id token", http.StatusInternalServerError)
		return
	}

	// Create access token (can be a simple random string or a JWT).
	accessToken := "some-access-token"

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"id_token":     idToken,
		"access_token": accessToken,
		"token_type":   "Bearer",
	})
}

func userinfoHandler(w http.ResponseWriter, r *http.Request) {
	// In a real implementation, you would validate the access token.
	// For simplicity, we'll just return some user info.
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"sub":   "user",
		"email": "user@example.com",
	})
}
