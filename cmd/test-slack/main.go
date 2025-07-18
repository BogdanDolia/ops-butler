package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
)

type TestMessage struct {
	Channel string `json:"channel"`
	Message string `json:"message"`
	TaskID  *uint  `json:"task_id,omitempty"`
}

func main() {
	apiURL := os.Getenv("API_URL")
	if apiURL == "" {
		apiURL = "http://localhost:8080"
	}

	// Test simple message
	testSimpleMessage(apiURL)

	// Test reminder message
	testReminderMessage(apiURL)
}

func testSimpleMessage(apiURL string) {
	msg := TestMessage{
		Channel: "#general",
		Message: "🤖 Hello from Ops Butler test script!",
	}

	sendTestMessage(apiURL, msg, "simple")
}

func testReminderMessage(apiURL string) {
	taskID := uint(42)
	msg := TestMessage{
		Channel: "#general",
		Message: "⏰ Task reminder: Please check the production logs",
		TaskID:  &taskID,
	}

	sendTestMessage(apiURL, msg, "reminder")
}

func sendTestMessage(apiURL string, msg TestMessage, msgType string) {
	jsonData, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Error marshaling %s message: %v", msgType, err)
		return
	}

	resp, err := http.Post(
		fmt.Sprintf("%s/api/v1/chatops/test/slack", apiURL),
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		log.Printf("Error sending %s message: %v", msgType, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		fmt.Printf("✅ %s message sent successfully\n", msgType)
	} else {
		fmt.Printf("❌ Failed to send %s message. Status: %d\n", msgType, resp.StatusCode)
	}
}
