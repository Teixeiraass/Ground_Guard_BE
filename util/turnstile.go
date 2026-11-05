package util

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"time"
)

const turnstileVerifyURL = "https://challenges.cloudflare.com/turnstile/v0/siteverify"

type turnstileResponse struct {
	Success bool `json:"success"`
}

// VerifyTurnstile checks a Cloudflare Turnstile response token against Cloudflare's
// siteverify endpoint. If no secret key is configured, verification is skipped and
// treated as successful, so the feature keeps working in local/dev setups that
// haven't configured Turnstile yet.
func VerifyTurnstile(ctx context.Context, secretKey, responseToken, remoteIP string) (bool, error) {
	if secretKey == "" {
		return true, nil
	}

	if responseToken == "" {
		return false, nil
	}

	form := url.Values{}
	form.Set("secret", secretKey)
	form.Set("response", responseToken)
	if remoteIP != "" {
		form.Set("remoteip", remoteIP)
	}

	httpCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(httpCtx, http.MethodPost, turnstileVerifyURL, nil)
	if err != nil {
		return false, err
	}
	req.URL.RawQuery = form.Encode()

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	var result turnstileResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, err
	}

	return result.Success, nil
}
