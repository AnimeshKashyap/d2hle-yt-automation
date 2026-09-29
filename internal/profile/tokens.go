package profile

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/d2hle/yt-automation/internal/domain"
)

func NewAPIToken(label string) (domain.APIToken, string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return domain.APIToken{}, "", err
	}
	plain := hex.EncodeToString(raw)
	sum := sha256.Sum256([]byte(plain))
	return domain.APIToken{
		ID:        fmt.Sprintf("tok_%s", hex.EncodeToString(sum[:8])),
		Label:     label,
		Hash:      hex.EncodeToString(sum[:]),
		CreatedAt: time.Now().UTC(),
	}, plain, nil
}

func TokenMatches(plain string, hash string) bool {
	sum := sha256.Sum256([]byte(plain))
	return hex.EncodeToString(sum[:]) == hash
}
