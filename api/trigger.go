package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

type sentryRequest struct {
	Data struct {
		Event struct {
			Title string `json:"title"`
			Url   string `json:"web_url"`
		} `json:"event"`
	} `json:"data"`
}

func Handler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if r.Header.Get("Authorization") != "Bearer "+os.Getenv("SENTRY_TOKEN") {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var sb sentryRequest
	if err := json.NewDecoder(r.Body).Decode(&sb); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	res, err := createPage(sb.Data.Event.Title, "New Sentry issue: "+sb.Data.Event.Url)
	if err != nil {
		log.Println(err)
		http.Error(w, "Notion request failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	fmt.Fprintf(w, "{\"ok\": true, \"message\": %s}", res)
}

func createPage(title, content string) ([]byte, error) {
	payload := map[string]any{
		"parent": map[string]any{
			"data_source_id": os.Getenv("NOTION_DATABASE"),
		},
		"properties": map[string]any{
			"Task": map[string]any{
				"title": []any{map[string]any{"text": map[string]any{"content": title}}},
			},
			"Type":     map[string]any{"select": map[string]any{"name": "Bug"}},
			"Status":   map[string]any{"status": map[string]any{"name": "Todo"}},
			"Priority": map[string]any{"select": map[string]any{"name": "P2"}},
		},
		"markdown": content,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", os.Getenv("NOTION_URL")+"pages", bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", "Bearer "+os.Getenv("NOTION_API_KEY"))
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Notion-Version", "2026-03-11")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}
