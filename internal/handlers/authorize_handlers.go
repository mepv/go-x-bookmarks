package handlers

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/mepv/go-x-bookmarks/internal/util"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"sync"
)

var pkceStorage = sync.Map{}

func BuildAuthorizationUrl(w http.ResponseWriter, r *http.Request) {
	authorizationUri := os.Getenv("AUTHORIZATION_URI")
	u, err := url.Parse(authorizationUri)
	if err != nil {
		log.Printf("Error parsing authorization url: %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	codeVerifier, err := util.GenerateCodeVerifier()
	if err != nil {
		log.Printf("Error generating code verifier: %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	codeChallenge := util.GenerateCodeChallenge(codeVerifier)
	state := uuid.New().String()
	storePKCE(state, codeVerifier)

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

	codeVerifier, _ := retrievePKCE(queryParams.Get("state"))
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
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Basic "+encodeCredentials(clientId, clientSecret))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error exchanging code for token: %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
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
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
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

	//todo: change to a custom page and redirect to it. The page to start interacting with the API, and display a message saying "Success"
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode("Success")
	log.Print("Successful exchanged code for access token")
}

func storePKCE(state string, codeVerifier string) {
	pkceStorage.Store(state, codeVerifier)
}

func retrievePKCE(state string) (string, bool) {
	var pkce, _ = pkceStorage.LoadAndDelete(state)
	return pkce.(string), true
}

func encodeCredentials(clientId, clientSecret string) string {
	credentials := clientId + ":" + clientSecret
	return base64.StdEncoding.EncodeToString([]byte(credentials))
}
