package handlers

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/mepv/go-x-bookmarks/internal/config"
	"github.com/mepv/go-x-bookmarks/internal/helpers"
	"github.com/mepv/go-x-bookmarks/internal/models"
	"github.com/mepv/go-x-bookmarks/internal/render"
	"github.com/mepv/go-x-bookmarks/internal/util"
)

const accessTokenNotFound = "Access token not found. Please signup again."

func NewAuthorizationHandlers(a *config.AppConfig) {
	app = a
}

func BuildAuthorizationUrl(w http.ResponseWriter, r *http.Request) {
	log.Printf("🔐 Starting OAuth authorization flow")
	
	authorizationUri := os.Getenv("AUTHORIZATION_URI")
	clientId := os.Getenv("CLIENT_ID")
	redirectUri := os.Getenv("REDIRECT_URI")
	scope := os.Getenv("SCOPE")
	
	log.Printf("🔐 OAuth Config - URI: %s, ClientID: %s, RedirectURI: %s, Scope: %s", 
		authorizationUri, maskSecret(clientId), redirectUri, scope)
	
	u, err := url.Parse(authorizationUri)
	if err != nil {
		log.Printf("❌ Error parsing authorization url: %v", err)
		helpers.ServerError(w, err)
		return
	}

	codeVerifier, err := util.GenerateCodeVerifier()
	if err != nil {
		log.Printf("❌ Error generating code verifier: %v", err)
		helpers.ServerError(w, err)
		return
	}
	codeChallenge := util.GenerateCodeChallenge(codeVerifier)
	state := uuid.New().String()
	
	log.Printf("🔐 Generated - State: %s, CodeVerifier: %s..., CodeChallenge: %s...", 
		state, codeVerifier[:10], codeChallenge[:10])

	err = app.Session.Store.Commit(state, []byte(codeVerifier), time.Now().Add(12*time.Hour))
	if err != nil {
		log.Printf("❌ Error committing state in session: %v", err)
		helpers.ServerError(w, err)
		return
	}
	log.Printf("✅ Successfully stored state in session")

	params := url.Values{}
	params.Add("response_type", "code")
	params.Add("client_id", clientId)
	params.Add("redirect_uri", redirectUri)
	params.Add("scope", scope)
	params.Add("state", state)
	params.Add("code_challenge", codeChallenge)
	params.Add("code_challenge_method", "S256")
	u.RawQuery = params.Encode()

	authUrl := u.String()
	log.Printf("🔗 Redirecting to authorization URL: %s", authUrl)
	http.Redirect(w, r, authUrl, http.StatusFound)
}

func ExchangeCodeForToken(w http.ResponseWriter, r *http.Request) {
	log.Printf("🔄 Starting token exchange callback")
	
	tokenUri := os.Getenv("TOKEN_URI")
	clientId := os.Getenv("CLIENT_ID")
	clientSecret := os.Getenv("CLIENT_SECRET")
	redirectUri := os.Getenv("REDIRECT_URI")
	
	queryParams := r.URL.Query()
	
	// Log all query parameters received from callback
	log.Printf("🔄 Callback URL: %s", r.URL.String())
	for key, values := range queryParams {
		log.Printf("🔄 Query param - %s: %s", key, values)
	}

	// Check for error in callback
	if errorParam := queryParams.Get("error"); errorParam != "" {
		errorDescription := queryParams.Get("error_description")
		log.Printf("❌ OAuth error received - Error: %s, Description: %s", errorParam, errorDescription)
		http.Error(w, fmt.Sprintf("OAuth error: %s - %s", errorParam, errorDescription), http.StatusBadRequest)
		return
	}

	state := queryParams.Get("state")
	code := queryParams.Get("code")
	
	codePreview := code
	if len(code) > 10 {
		codePreview = code[:10] + "..."
	}
	log.Printf("🔄 Received - State: %s, Code: %s", state, codePreview)
	
	if state == "" {
		log.Printf("❌ No state parameter received")
		http.Error(w, "Missing state parameter", http.StatusBadRequest)
		return
	}
	
	if code == "" {
		log.Printf("❌ No authorization code received")
		http.Error(w, "Missing authorization code", http.StatusBadRequest)
		return
	}

	// Look up the code verifier from session
	codeVerifierData, found, err := app.Session.Store.Find(state)
	if err != nil {
		log.Printf("❌ Error finding state in session: %v", err)
		helpers.ServerError(w, err)
		return
	}
	
	if !found {
		log.Printf("❌ State not found in session: %s", state)
		http.Error(w, "Invalid state parameter", http.StatusBadRequest)
		return
	}

	// Delete the state from session
	err = app.Session.Store.Delete(state)
	if err != nil {
		log.Printf("⚠️ Error deleting state from session: %v", err)
		// Continue anyway, this isn't critical
	}

	codeVerifier := string(codeVerifierData)
	if codeVerifier == "" {
		log.Printf("❌ No code verifier found for state: %s", state)
		http.Error(w, "Invalid session state", http.StatusInternalServerError)
		return
	}
	
	verifierPreview := codeVerifier
	if len(codeVerifier) > 10 {
		verifierPreview = codeVerifier[:10] + "..."
	}
	log.Printf("✅ Found code verifier: %s", verifierPreview)

	params := url.Values{}
	params.Add("grant_type", "authorization_code")
	params.Add("client_id", clientId)
	params.Add("client_secret", clientSecret)
	params.Add("code", code)
	params.Add("redirect_uri", redirectUri)
	params.Add("code_verifier", codeVerifier)
	
	log.Printf("🔄 Token request params - grant_type: %s, client_id: %s, redirect_uri: %s", 
		"authorization_code", maskSecret(clientId), redirectUri)

	req, err := http.NewRequest("POST", tokenUri, bytes.NewBufferString(params.Encode()))
	if err != nil {
		log.Printf("❌ Error creating token request: %v", err)
		helpers.ServerError(w, err)
		return
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Basic "+encodeCredentials(clientId, clientSecret))
	
	log.Printf("🔄 Sending token request to: %s", tokenUri)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("❌ Error sending token request: %v", err)
		helpers.ServerError(w, err)
		return
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Printf("⚠️ Error closing response body: %v", err)
		}
	}(resp.Body)

	log.Printf("🔄 Token response - Status: %d %s", resp.StatusCode, resp.Status)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("❌ Error reading token response body: %v", err)
		helpers.ServerError(w, err)
		return
	}

	log.Printf("🔄 Token response body: %s", string(body))

	if resp.StatusCode != http.StatusOK {
		log.Printf("❌ Token request failed with status %d", resp.StatusCode)
		
		// Attempt to parse error details from the response
		var errResp map[string]interface{}
		if err := json.Unmarshal(body, &errResp); err != nil {
			log.Printf("❌ Error parsing error response JSON: %v", err)
			http.Error(w, fmt.Sprintf("Authorization failed (HTTP %d)", resp.StatusCode), http.StatusUnauthorized)
			return
		}

		log.Printf("❌ Parsed error response: %+v", errResp)

		// Respond with the error message from the token endpoint
		errorMessage, exists := errResp["error_description"].(string)
		if !exists {
			if errorType, ok := errResp["error"].(string); ok {
				errorMessage = fmt.Sprintf("Authorization failed: %s", errorType)
			} else {
				errorMessage = fmt.Sprintf("Authorization failed (HTTP %d)", resp.StatusCode)
			}
		}

		log.Printf("❌ Final error message: %s", errorMessage)
		http.Error(w, errorMessage, http.StatusUnauthorized)
		return
	}

	log.Printf("✅ Successfully exchanged code for access token!")

	var tokenResp models.TokenResponse
	err = json.Unmarshal(body, &tokenResp)
	if err != nil {
		log.Printf("❌ Failed to decode token response JSON: %v", err)
		helpers.ServerError(w, err)
		return
	}
	
	log.Printf("✅ Token response parsed - TokenType: %s, ExpiresIn: %d, Scope: %s", 
		tokenResp.TokenType, tokenResp.ExpiresIn, tokenResp.Scope)
	
	// Store access token in session
	app.Session.Put(r.Context(), "access_token", tokenResp.AccessToken)
	log.Printf("✅ Access token stored in session successfully")

	data := make(map[string]interface{})
	data["success"] = "🎉 Authorization successful! You can now fetch your bookmarks."
	err = render.Template(w, r, "user.page.gohtml", &models.TemplateData{
		Data: data,
	})
	if err != nil {
		log.Printf("❌ Error rendering success template: %v", err)
		helpers.ServerError(w, err)
		return
	}
	
	log.Printf("✅ OAuth flow completed successfully!")
}

func encodeCredentials(clientId, clientSecret string) string {
	credentials := clientId + ":" + clientSecret
	return base64.StdEncoding.EncodeToString([]byte(credentials))
}

// maskSecret returns first 4 chars + "..." for logging sensitive data safely
func maskSecret(secret string) string {
	if len(secret) <= 4 {
		return "***"
	}
	return secret[:4] + "..."
}
