package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"

	"github.com/gorilla/websocket"
	"github.com/telnet2/go-practice/go-memsh/api"
)

func main() {
	serverURL := flag.String("server", "http://localhost:8080", "API server URL")
	flag.Parse()

	// Parse server URL
	u, err := url.Parse(*serverURL)
	if err != nil {
		log.Fatalf("Invalid server URL: %v", err)
	}

	// Create a new session
	fmt.Println("Creating new session...")
	session, err := createSession(*serverURL)
	if err != nil {
		log.Fatalf("Failed to create session: %v", err)
	}

	fmt.Printf("Session created: %s\n", session.ID)
	fmt.Printf("Current directory: %s\n\n", session.Cwd)

	// Connect to WebSocket REPL
	wsURL := fmt.Sprintf("ws://%s/api/v1/session/repl", u.Host)
	fmt.Printf("Connecting to REPL at %s...\n", wsURL)

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		log.Fatalf("Failed to connect to WebSocket: %v", err)
	}
	defer conn.Close()

	fmt.Println("Connected! Running example commands...\n")

	// Example commands to execute
	commands := []struct {
		command string
		args    []string
		desc    string
	}{
		{"pwd", nil, "Show current directory"},
		{"mkdir", []string{"-p", "/home/user"}, "Create directory"},
		{"cd", []string{"/home/user"}, "Change directory"},
		{"pwd", nil, "Verify directory change"},
		{"echo", []string{"Hello from API!"}, "Echo message"},
		{"echo", []string{"'Test data'"}, "Create test file"},
		{"ls", []string{"-la"}, "List current directory"},
	}

	for i, cmd := range commands {
		fmt.Printf("[%d] %s: %s %v\n", i+1, cmd.desc, cmd.command, cmd.args)

		result, err := executeCommand(conn, session.ID, cmd.command, cmd.args)
		if err != nil {
			fmt.Printf("Error: %v\n\n", err)
			continue
		}

		if result.Error != "" {
			fmt.Printf("Command error: %s\n", result.Error)
		}

		if len(result.Output) > 0 {
			fmt.Println("Output:")
			for _, line := range result.Output {
				fmt.Printf("  %s\n", line)
			}
		}

		fmt.Printf("Current directory: %s\n\n", result.Cwd)
	}

	// List all sessions
	fmt.Println("Listing all sessions...")
	sessions, err := listSessions(*serverURL)
	if err != nil {
		log.Fatalf("Failed to list sessions: %v", err)
	}

	fmt.Printf("Total sessions: %d\n", len(sessions))
	for _, s := range sessions {
		fmt.Printf("  - %s (cwd: %s)\n", s.ID, s.Cwd)
	}

	// Remove session
	fmt.Printf("\nRemoving session %s...\n", session.ID)
	err = removeSession(*serverURL, session.ID)
	if err != nil {
		log.Fatalf("Failed to remove session: %v", err)
	}

	fmt.Println("Session removed successfully!")
}

func createSession(baseURL string) (*api.SessionInfo, error) {
	resp, err := http.Post(baseURL+"/api/v1/session/create", "application/json", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var result api.CreateSessionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result.Session, nil
}

func listSessions(baseURL string) ([]api.SessionInfo, error) {
	resp, err := http.Post(baseURL+"/api/v1/session/list", "application/json", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var result api.ListSessionsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Sessions, nil
}

func removeSession(baseURL, sessionID string) error {
	reqBody := api.RemoveSessionRequest{SessionID: sessionID}
	bodyBytes, _ := json.Marshal(reqBody)

	resp, err := http.Post(baseURL+"/api/v1/session/remove", "application/json", bytes.NewReader(bodyBytes))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

func executeCommand(conn *websocket.Conn, sessionID, command string, args []string) (*api.ExecuteCommandResult, error) {
	// Create JSON-RPC request
	request := api.JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "shell.execute",
		Params: json.RawMessage(mustMarshal(api.ExecuteCommandParams{
			SessionID: sessionID,
			Command:   command,
			Args:      args,
		})),
		ID: 1,
	}

	// Send request
	if err := conn.WriteJSON(request); err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	// Read response
	var response api.JSONRPCResponse
	if err := conn.ReadJSON(&response); err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check for errors
	if response.Error != nil {
		return nil, fmt.Errorf("JSON-RPC error: %s", response.Error.Message)
	}

	// Parse result
	resultBytes, err := json.Marshal(response.Result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	var result api.ExecuteCommandResult
	if err := json.Unmarshal(resultBytes, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result: %w", err)
	}

	return &result, nil
}

func mustMarshal(v interface{}) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to marshal: %v\n", err)
		os.Exit(1)
	}
	return data
}
