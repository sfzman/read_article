package inference

import (
	"bytes"
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"time"

	"read_article/backend/internal/config"

	"github.com/golang-jwt/jwt/v5"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
	privateKey *rsa.PrivateKey
	expiresIn  time.Duration
}

type TTSRequest struct {
	Text           string    `json:"text"`
	ReferenceAudio string    `json:"reference_audio,omitempty"`
	EmotionPrompt  string    `json:"emotion_prompt,omitempty"`
	EmotionVector  []float64 `json:"emotion_vector,omitempty"`
	EmotionAlpha   *float64  `json:"emotion_alpha,omitempty"`
	UseEmotionText *bool     `json:"use_emotion_text,omitempty"`
}

func NewClient(cfg config.Config) (*Client, error) {
	client := &Client{
		baseURL: cfg.InferenceURL,
		httpClient: &http.Client{
			Timeout: cfg.RequestTimeout,
		},
		expiresIn: time.Duration(cfg.JWTExpireSeconds) * time.Second,
	}

	if cfg.JWTPrivateKey == "" {
		return client, nil
	}

	block, _ := pem.Decode([]byte(cfg.JWTPrivateKey))
	if block == nil {
		return nil, fmt.Errorf("decode JWT private key: invalid PEM")
	}

	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		key, err = x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("parse JWT private key: %w", err)
		}
	}

	privateKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("JWT private key is not RSA")
	}

	client.privateKey = privateKey
	return client, nil
}

func (c *Client) Generate(ctx context.Context, req TTSRequest) ([]byte, error) {
	payload, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal inference request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v1/tts", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("create inference request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	token, err := c.generateJWT()
	if err != nil {
		return nil, err
	}
	if token != "" {
		httpReq.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("call inference service: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read inference response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("inference service returned %d: %s", resp.StatusCode, string(body))
	}

	return body, nil
}

func (c *Client) generateJWT() (string, error) {
	if c.privateKey == nil {
		return "", nil
	}

	now := time.Now()
	claims := jwt.MapClaims{
		"iat": now.Unix(),
		"exp": now.Add(c.expiresIn).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	signed, err := token.SignedString(c.privateKey)
	if err != nil {
		return "", fmt.Errorf("sign JWT: %w", err)
	}
	return signed, nil
}
