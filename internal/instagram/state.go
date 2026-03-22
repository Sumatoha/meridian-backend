package instagram

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

type oauthState struct {
	UserID    uuid.UUID  `json:"u"`
	AccountID *uuid.UUID `json:"a,omitempty"`
}

// EncodeState creates a signed, base64-encoded state parameter for OAuth.
func EncodeState(userID uuid.UUID, accountID *uuid.UUID, secret string) (string, error) {
	state := oauthState{UserID: userID, AccountID: accountID}

	payload, err := json.Marshal(state)
	if err != nil {
		return "", fmt.Errorf("marshal state: %w", err)
	}

	sig := signPayload(payload, secret)
	payloadB64 := base64.RawURLEncoding.EncodeToString(payload)
	sigB64 := base64.RawURLEncoding.EncodeToString(sig)

	return payloadB64 + "." + sigB64, nil
}

// DecodeState verifies and decodes the OAuth state parameter.
func DecodeState(state, secret string) (uuid.UUID, *uuid.UUID, error) {
	parts := strings.SplitN(state, ".", 2)
	if len(parts) != 2 {
		return uuid.Nil, nil, fmt.Errorf("invalid state format")
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return uuid.Nil, nil, fmt.Errorf("decode state payload: %w", err)
	}

	sig, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return uuid.Nil, nil, fmt.Errorf("decode state signature: %w", err)
	}

	expectedSig := signPayload(payload, secret)
	if !hmac.Equal(sig, expectedSig) {
		return uuid.Nil, nil, fmt.Errorf("invalid state signature")
	}

	var s oauthState
	if err := json.Unmarshal(payload, &s); err != nil {
		return uuid.Nil, nil, fmt.Errorf("unmarshal state: %w", err)
	}

	return s.UserID, s.AccountID, nil
}

func signPayload(payload []byte, secret string) []byte {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return mac.Sum(nil)
}
