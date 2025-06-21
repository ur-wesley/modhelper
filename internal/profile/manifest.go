package profile

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/ur-wesley/modhelper/internal"
)

func FetchGames(manifestURL string) ([]internal.Game, error) {
	separator := "?"
	if strings.Contains(manifestURL, "?") {
		separator = "&"
	}
	timestampedURL := fmt.Sprintf("%s%st=%d", manifestURL, separator, time.Now().Unix())

	log.Printf("Fetching manifest from: %s", timestampedURL)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(timestampedURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch manifest: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("manifest request failed with status: %s", resp.Status)
	}

	var games []internal.Game
	if err := json.NewDecoder(resp.Body).Decode(&games); err != nil {
		return nil, fmt.Errorf("failed to parse manifest: %w", err)
	}

	return games, nil
}
