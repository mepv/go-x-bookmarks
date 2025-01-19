package handlers

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/mepv/go-x-bookmarks/internal/config"
	"github.com/mepv/go-x-bookmarks/internal/helpers"
	"github.com/mepv/go-x-bookmarks/internal/models"
	"github.com/mepv/go-x-bookmarks/internal/render"
	"github.com/mepv/go-x-bookmarks/internal/util"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"
)

const accessTokenNotFound = "Access token not found. Please signup again."

var pkceStorage = sync.Map{}

func NewAuthorizationHandlers(a *config.AppConfig) {
	app = a
}

func BuildAuthorizationUrl(w http.ResponseWriter, r *http.Request) {
	authorizationUri := os.Getenv("AUTHORIZATION_URI")
	u, err := url.Parse(authorizationUri)
	if err != nil {
		log.Printf("Error parsing authorization url: %v", err)
		helpers.ServerError(w, err)
		return
	}

	codeVerifier, err := util.GenerateCodeVerifier()
	if err != nil {
		log.Printf("Error generating code verifier: %v", err)
		helpers.ServerError(w, err)
		return
	}
	codeChallenge := util.GenerateCodeChallenge(codeVerifier)
	state := uuid.New().String()

	err = app.Session.Store.Commit(state, []byte(codeVerifier), time.Now().Add(12*time.Hour))
	if err != nil {
		log.Printf("Error committing state in session: %v", err)
		helpers.ServerError(w, err)
		return
	}

	params := url.Values{}
	params.Add("response_type", "code")
	params.Add("client_id", os.Getenv("CLIENT_ID"))
	params.Add("redirect_uri", os.Getenv("REDIRECT_URI"))
	params.Add("scope", os.Getenv("SCOPE"))
	params.Add("state", state)
	params.Add("code_challenge", codeChallenge)
	params.Add("code_challenge_method", "S256")
	u.RawQuery = params.Encode()

	http.Redirect(w, r, u.String(), http.StatusFound)
	log.Printf("Building and redirecting to authorization URL")
}

func ExchangeCodeForToken(w http.ResponseWriter, r *http.Request) {
	tokenUri := os.Getenv("TOKEN_URI")
	clientId := os.Getenv("CLIENT_ID")
	clientSecret := os.Getenv("CLIENT_SECRET")
	queryParams := r.URL.Query()

	state := queryParams.Get("state")
	codeVerifierData, _, err := app.Session.Store.Find(state)
	err = app.Session.Store.Delete(state)
	if err != nil {
		log.Printf("Error deleting state from session: %v", err)
		helpers.ServerError(w, err)
		return
	}

	codeVerifier := string(codeVerifierData)
	if codeVerifier == "" {
		log.Printf("No code verifier found")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	params := url.Values{}
	params.Add("grant_type", "authorization_code")
	params.Add("client_id", clientId)
	params.Add("client_secret", clientSecret)
	params.Add("code", queryParams.Get("code"))
	params.Add("redirect_uri", os.Getenv("REDIRECT_URI"))
	params.Add("code_verifier", codeVerifier)

	req, err := http.NewRequest("POST", tokenUri, bytes.NewBufferString(params.Encode()))
	if err != nil {
		log.Printf("Error creating request: %v", err)
		helpers.ServerError(w, err)
		return
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Basic "+encodeCredentials(clientId, clientSecret))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error exchanging code for token: %v", err)
		helpers.ServerError(w, err)
		return
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			//
		}
	}(resp.Body)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading response body: %v", err)
		helpers.ServerError(w, err)
		return
	}

	if resp.StatusCode != http.StatusOK {
		// Attempt to parse error details from the response
		var errResp map[string]interface{}
		if err := json.Unmarshal(body, &errResp); err != nil {
			log.Printf("Error parsing error response: %v", err)
			http.Error(w, "Authorization failed", http.StatusUnauthorized)
			return
		}

		// Respond with the error message from the token endpoint
		errorMessage, exists := errResp["error_description"].(string)
		if !exists {
			errorMessage = "Authorization failed"
		}

		http.Error(w, errorMessage, http.StatusUnauthorized)
		log.Printf("Token request failed: %s", errorMessage)
		return
	}

	log.Print("Successful exchanged code for access token")

	var tokenResp models.TokenResponse
	err = json.Unmarshal(body, &tokenResp)
	if err != nil {
		log.Printf("failed to decode token response: %v", err)
		helpers.ServerError(w, err)
		return
	}
	app.Session.Put(r.Context(), "access_token", tokenResp.AccessToken)

	data := make(map[string]interface{})
	data["success"] = "Success"
	err = render.Template(w, r, "user.page.gohtml", &models.TemplateData{
		Data: data,
	})
	if err != nil {
		log.Printf("Error rendering template: %v", err)
		helpers.ServerError(w, err)
		return
	}
}

func encodeCredentials(clientId, clientSecret string) string {
	credentials := clientId + ":" + clientSecret
	return base64.StdEncoding.EncodeToString([]byte(credentials))
}
