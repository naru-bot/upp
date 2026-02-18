package notify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"time"
)

type Event struct {
	Target   string `json:"target"`
	URL      string `json:"url"`
	Status   string `json:"status"`
	OldHash  string `json:"old_hash,omitempty"`
	NewHash  string `json:"new_hash,omitempty"`
	Error    string `json:"error,omitempty"`
	Time     string `json:"time"`
	Message  string `json:"message"`
}

func Send(typ, config string, event Event) error {
	switch typ {
	case "webhook":
		return sendWebhook(config, event)
	case "command":
		return sendCommand(config, event)
	case "slack":
		return sendSlack(config, event)
	case "telegram":
		return sendTelegram(config, event)
	case "discord":
		return sendDiscord(config, event)
	default:
		return fmt.Errorf("unknown notification type: %s", typ)
	}
}

func sendWebhook(configJSON string, event Event) error {
	var cfg struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil {
		return err
	}

	body, _ := json.Marshal(event)
	resp, err := http.Post(cfg.URL, "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook returned %d", resp.StatusCode)
	}
	return nil
}

func sendCommand(configJSON string, event Event) error {
	var cfg struct {
		Command string `json:"command"`
	}
	if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil {
		return err
	}

	cmdStr := cfg.Command
	cmdStr = strings.ReplaceAll(cmdStr, "{target}", event.Target)
	cmdStr = strings.ReplaceAll(cmdStr, "{url}", event.URL)
	cmdStr = strings.ReplaceAll(cmdStr, "{status}", event.Status)
	cmdStr = strings.ReplaceAll(cmdStr, "{message}", event.Message)

	cmd := exec.Command("sh", "-c", cmdStr)
	return cmd.Run()
}

func sendSlack(configJSON string, event Event) error {
	var cfg struct {
		WebhookURL string `json:"webhook_url"`
	}
	if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil {
		return err
	}

	payload := map[string]string{"text": event.Message}
	body, _ := json.Marshal(payload)
	resp, err := http.Post(cfg.WebhookURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

func sendTelegram(configJSON string, event Event) error {
	var cfg struct {
		BotToken string `json:"bot_token"`
		ChatID   string `json:"chat_id"`
	}
	if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil {
		return err
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", cfg.BotToken)
	payload := map[string]string{
		"chat_id": cfg.ChatID,
		"text":    event.Message,
	}
	body, _ := json.Marshal(payload)
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

func sendDiscord(configJSON string, event Event) error {
	var cfg struct {
		WebhookURL string `json:"webhook_url"`
	}
	if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil {
		return err
	}

	payload := map[string]string{"content": event.Message}
	body, _ := json.Marshal(payload)
	resp, err := http.Post(cfg.WebhookURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}
