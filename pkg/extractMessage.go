package pkg

import (
	"encoding/json"
	"fmt"
)

func ExtractMessage(fullMessage string, Extract string) (string, error) {
	if Extract == "message" {
		type Message struct {
			Message string `json:"message"`
		}
		var m Message
		err := json.Unmarshal([]byte(fullMessage), &m)
		if err != nil {
			return "", fmt.Errorf("Failed to extract message: %v", err)
		}
		return m.Message, nil
	} else if Extract == "userId" {
		type Message struct {
			UserID string `json:"userId"`
		}

		var m Message
		err := json.Unmarshal([]byte(fullMessage), &m)
		if err != nil {
			return "", fmt.Errorf("Failed to extract message: %v", err)
		}
		return m.UserID, nil
	}
	return "", nil
}
