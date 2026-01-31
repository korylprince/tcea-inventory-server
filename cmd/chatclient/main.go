package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/gorilla/websocket"
)

type authRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type authResponse struct {
	SessionKey string `json:"session_key"`
}

type clientMessage struct {
	Message string `json:"message"`
}

type serverMessage struct {
	Type           string `json:"type"`
	Content        string `json:"content,omitempty"`
	ConversationID string `json:"conversation_id,omitempty"`
	Error          string `json:"error,omitempty"`
}

func main() {
	server := flag.String("server", "http://localhost:8080", "Server URL (http/https)")
	email := flag.String("email", "", "User email for authentication")
	password := flag.String("password", "", "User password for authentication")
	conversationID := flag.String("conversation", "", "Conversation ID to continue (optional)")
	flag.Parse()

	if *email == "" || *password == "" {
		fmt.Println("Error: -email and -password are required")
		flag.Usage()
		os.Exit(1)
	}

	// Authenticate
	sessionKey, err := authenticate(*server, *email, *password)
	if err != nil {
		fmt.Printf("Authentication failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Authentication successful!")

	// Convert HTTP URL to WebSocket URL
	wsURL := strings.Replace(*server, "http://", "ws://", 1)
	wsURL = strings.Replace(wsURL, "https://", "wss://", 1)
	wsURL += "/api/1.0/chat"

	reader := bufio.NewReader(os.Stdin)
	currentConvID := *conversationID

	for {
		fmt.Print("\nYou: ")
		input, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				fmt.Println("\nGoodbye!")
				return
			}
			fmt.Printf("Error reading input: %v\n", err)
			continue
		}

		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}
		if strings.ToLower(input) == "exit" || strings.ToLower(input) == "quit" {
			fmt.Println("Goodbye!")
			return
		}

		// Connect to WebSocket
		url := wsURL
		if currentConvID != "" {
			url += "?conversation_id=" + currentConvID
		}

		header := http.Header{}
		header.Set("X-Session-Key", sessionKey)

		conn, _, err := websocket.DefaultDialer.Dial(url, header)
		if err != nil {
			fmt.Printf("WebSocket connection failed: %v\n", err)
			continue
		}

		// Send message
		if err := conn.WriteJSON(clientMessage{Message: input}); err != nil {
			fmt.Printf("Failed to send message: %v\n", err)
			conn.Close()
			continue
		}

		// Read response
		fmt.Print("Assistant: ")
		for {
			var msg serverMessage
			if err := conn.ReadJSON(&msg); err != nil {
				if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
					break
				}
				fmt.Printf("\nError reading response: %v\n", err)
				break
			}

			switch msg.Type {
			case "text":
				fmt.Print(msg.Content)
			case "done":
				fmt.Println()
				currentConvID = msg.ConversationID
				fmt.Printf("(Conversation ID: %s)\n", currentConvID)
			case "error":
				fmt.Printf("\nError: %s\n", msg.Error)
			}

			if msg.Type == "done" || msg.Type == "error" {
				break
			}
		}

		conn.Close()
	}
}

func authenticate(serverURL, email, password string) (string, error) {
	authReq := authRequest{Email: email, Password: password}
	body, err := json.Marshal(authReq)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := http.Post(serverURL+"/api/1.0/auth", "application/json", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("authentication failed (status %d): %s", resp.StatusCode, string(respBody))
	}

	var authResp authResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return authResp.SessionKey, nil
}
