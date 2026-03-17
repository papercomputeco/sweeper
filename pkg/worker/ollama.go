package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// OllamaConfig holds settings for the Ollama executor.
type OllamaConfig struct {
	Model   string // e.g. "qwen2.5-coder:7b"
	APIBase string // e.g. "http://localhost:11434"
}

// NewOllamaExecutor returns an Executor that calls the Ollama HTTP API.
// It sends the prompt as a chat message, extracts a unified diff from the
// response, and applies it via `patch -p1`.
func NewOllamaExecutor(cfg OllamaConfig) Executor {
	if cfg.APIBase == "" {
		cfg.APIBase = "http://localhost:11434"
	}
	if cfg.Model == "" {
		cfg.Model = "qwen2.5-coder:7b"
	}
	client := &http.Client{Timeout: 5 * time.Minute}

	return func(ctx context.Context, task Task) Result {
		start := time.Now()
		prompt := task.Prompt
		if prompt == "" {
			prompt = BuildAPIPrompt(task)
		}

		body, err := ollamaChat(ctx, client, cfg, prompt)
		duration := time.Since(start)
		if err != nil {
			return Result{
				TaskID: task.ID, File: task.File, Success: false,
				Output: body, Error: err.Error(), Duration: duration,
			}
		}

		diff := extractDiff(body)
		if diff == "" {
			return Result{
				TaskID: task.ID, File: task.File, Success: false,
				Output: body, Error: "no diff block found in response", Duration: duration,
			}
		}

		if err := applyDiff(task.Dir, diff); err != nil {
			return Result{
				TaskID: task.ID, File: task.File, Success: false,
				Output: body, Error: fmt.Sprintf("patch failed: %v", err), Duration: duration,
			}
		}

		return Result{
			TaskID: task.ID, File: task.File, Success: true,
			Output: body, Duration: duration, IssuesFix: len(task.Issues),
		}
	}
}

type ollamaChatRequest struct {
	Model    string          `json:"model"`
	Messages []ollamaMessage `json:"messages"`
	Stream   bool            `json:"stream"`
}

type ollamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ollamaChatResponse struct {
	Message ollamaMessage `json:"message"`
}

func ollamaChat(ctx context.Context, client *http.Client, cfg OllamaConfig, prompt string) (string, error) {
	reqBody := ollamaChatRequest{
		Model: cfg.Model,
		Messages: []ollamaMessage{
			{Role: "system", Content: agentPreamble},
			{Role: "user", Content: prompt},
		},
		Stream: false,
	}
	data, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshaling request: %w", err)
	}

	url := strings.TrimRight(cfg.APIBase, "/") + "/api/chat"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("calling ollama: %w", err)
	}
	defer resp.Body.Close()

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return string(respData), fmt.Errorf("ollama returned %d: %s", resp.StatusCode, string(respData))
	}

	var chatResp ollamaChatResponse
	if err := json.Unmarshal(respData, &chatResp); err != nil {
		return string(respData), fmt.Errorf("parsing response: %w", err)
	}

	return chatResp.Message.Content, nil
}

var diffBlockRe = regexp.MustCompile("(?s)```diff\\s*\n(.*?)```")

// extractDiff pulls the first unified diff from ```diff ... ``` markers.
func extractDiff(response string) string {
	m := diffBlockRe.FindStringSubmatch(response)
	if len(m) < 2 {
		return ""
	}
	return strings.TrimSpace(m[1])
}

// applyDiff writes the diff to a temp file and runs `patch -p1` in the target dir.
func applyDiff(dir, diff string) error {
	tmp, err := os.CreateTemp("", "sweeper-patch-*.diff")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())

	if _, err := tmp.WriteString(diff); err != nil {
		tmp.Close()
		return err
	}
	tmp.Close()

	absDir, err := filepath.Abs(dir)
	if err != nil {
		return err
	}

	cmd := exec.Command("patch", "-p1", "--no-backup-if-mismatch", "-i", tmp.Name())
	cmd.Dir = absDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %w", strings.TrimSpace(string(out)), err)
	}
	return nil
}
