package notification

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/google/uuid"
)

type sendResult struct {
	MessageID string
}

func (s *Service) send(ctx context.Context, provider, phone, body string) (sendResult, error) {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "", "noop":
		return sendResult{MessageID: "noop-" + uuid.NewString()}, nil
	case "fonnte":
		return s.sendFonnte(ctx, phone, body)
	case "wablas":
		return s.sendWablas(ctx, phone, body)
	default:
		return sendResult{}, fmt.Errorf("provider WhatsApp tidak dikenal: %s", provider)
	}
}

// sendFonnte delivers a message through the Fonnte gateway.
func (s *Service) sendFonnte(ctx context.Context, phone, body string) (sendResult, error) {
	token := strings.TrimSpace(s.deps.FonnteToken)
	if token == "" {
		return sendResult{}, errors.New("token Fonnte belum dikonfigurasi")
	}
	form := url.Values{}
	form.Set("target", phone)
	form.Set("message", body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.fonnte.com/send", strings.NewReader(form.Encode()))
	if err != nil {
		return sendResult{}, err
	}
	req.Header.Set("Authorization", token)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	result, err := s.doJSON(req)
	if err != nil {
		return sendResult{}, err
	}
	var payload struct {
		Status bool            `json:"status"`
		ID     json.RawMessage `json:"id"`
		Reason string          `json:"reason"`
	}
	if err := json.Unmarshal(result, &payload); err != nil {
		return sendResult{}, fmt.Errorf("respons Fonnte tidak valid: %w", err)
	}
	if !payload.Status {
		return sendResult{}, fmt.Errorf("Fonnte menolak pesan: %s", strings.TrimSpace(payload.Reason))
	}
	return sendResult{MessageID: firstJSONID(payload.ID)}, nil
}

// sendWablas delivers a message through the Wablas gateway.
func (s *Service) sendWablas(ctx context.Context, phone, body string) (sendResult, error) {
	token := strings.TrimSpace(s.deps.WablasToken)
	if token == "" {
		return sendResult{}, errors.New("token Wablas belum dikonfigurasi")
	}
	base := strings.TrimSpace(s.deps.WablasURL)
	if base == "" {
		base = "https://pati.wablas.com"
	}
	payload, err := json.Marshal(map[string]string{"phone": phone, "message": body})
	if err != nil {
		return sendResult{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(base, "/")+"/api/send-message", bytes.NewReader(payload))
	if err != nil {
		return sendResult{}, err
	}
	auth := token
	if secret := strings.TrimSpace(s.deps.WablasSecret); secret != "" {
		auth = token + "." + secret
	}
	req.Header.Set("Authorization", auth)
	req.Header.Set("Content-Type", "application/json")
	result, err := s.doJSON(req)
	if err != nil {
		return sendResult{}, err
	}
	var response struct {
		Status bool            `json:"status"`
		Data   json.RawMessage `json:"data"`
		Reason string          `json:"reason"`
	}
	if err := json.Unmarshal(result, &response); err != nil {
		return sendResult{}, fmt.Errorf("respons Wablas tidak valid: %w", err)
	}
	if !response.Status {
		return sendResult{}, fmt.Errorf("Wablas menolak pesan: %s", strings.TrimSpace(response.Reason))
	}
	var data struct {
		ID json.RawMessage `json:"id"`
	}
	_ = json.Unmarshal(response.Data, &data)
	return sendResult{MessageID: firstJSONID(data.ID)}, nil
}

func (s *Service) doJSON(req *http.Request) ([]byte, error) {
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("gateway WhatsApp merespons HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return body, nil
}

func firstJSONID(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		return text
	}
	var list []string
	if err := json.Unmarshal(raw, &list); err == nil && len(list) > 0 {
		return list[0]
	}
	var listAny []any
	if err := json.Unmarshal(raw, &listAny); err == nil && len(listAny) > 0 {
		return fmt.Sprint(listAny[0])
	}
	return strings.TrimSpace(string(raw))
}
