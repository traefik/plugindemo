package main

import (
    "context"
    "crypto/hmac"
    "crypto/sha256"
    "encoding/base64"
    "fmt"
    "net/http"
)

// Config the plugin configuration.
type Config struct {
    Headers   []string `json:"headers,omitempty"` // Headers to include in the signature computation
    SecretKey string   `json:"secretKey,omitempty"`
}

// CreateConfig creates the default plugin configuration.
func CreateConfig() *Config {
    return &Config{
        Headers:   []string{"X-Date", "Authorization", "APP-ID"},
        SecretKey: "",
    }
}

// SignatureVerifier a plugin to verify request signatures.
type SignatureVerifier struct {
    next     http.Handler
    headers  []string
    secretKey string
}

// New creates a new SignatureVerifier plugin.
func New(ctx context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
    if len(config.Headers) == 0 || config.SecretKey == "" {
        return nil, fmt.Errorf("headers and secretKey must be provided")
    }
    return &SignatureVerifier{
        next:     next,
        headers:  config.Headers,
        secretKey: config.SecretKey,
    }, nil
}

func (sv *SignatureVerifier) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
    // Extract required headers
    var messageParts []string
    for _, header := range sv.headers {
        value := req.Header.Get(header)
        if value == "" {
            http.Error(rw, fmt.Sprintf("Missing required header: %s", header), http.StatusForbidden)
            return
        }
        messageParts = append(messageParts, value)
    }

    // Generate the expected signature
    message := fmt.Sprint(messageParts...)
    expectedSignature := base64.StdEncoding.EncodeToString(
        hmac.New(sha256.New, []byte(sv.secretKey)).Sum([]byte(message)),
    )

    // Verify the X-Signature header
    signature := req.Header.Get("X-Signature")
    if signature != expectedSignature {
        http.Error(rw, "Invalid signature", http.StatusForbidden)
        return
    }

    // If all checks pass, forward the request to the next handler
    sv.next.ServeHTTP(rw, req)
}
