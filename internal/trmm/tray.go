package trmm

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/example/gotray/internal/config"
	"github.com/example/gotray/internal/logging"
)

// TrayData represents Tactical RMM custom field data mapped to GoTray constructs.
type TrayData struct {
	MenuItems []config.MenuItem
	Icon      []byte
}

var errNotFound = errors.New("tacticalrmm: not found")

type customFieldDefinition struct {
	ID            int      `json:"id"`
	Model         string   `json:"model"`
	Name          string   `json:"name"`
	Type          string   `json:"type"`
	DefaultString *string  `json:"default_value_string"`
	DefaultBool   *bool    `json:"default_value_bool"`
	DefaultValues []string `json:"default_values_multiple"`
}

func (d customFieldDefinition) Default() string {
	switch strings.ToLower(d.Type) {
	case "checkbox":
		if d.DefaultBool != nil {
			if *d.DefaultBool {
				return "true"
			}
			return "false"
		}
	case "multiple":
		if len(d.DefaultValues) > 0 {
			return strings.Join(d.DefaultValues, ",")
		}
	}
	if d.DefaultString != nil {
		return *d.DefaultString
	}
	return ""
}

type customFieldValue struct {
	Field         json.RawMessage `json:"field"`
	Value         json.RawMessage `json:"value"`
	StringValue   *string         `json:"string_value"`
	BoolValue     *bool           `json:"bool_value"`
	MultipleValue []string        `json:"multiple_value"`
}

func (v customFieldValue) FieldID() int {
	if len(v.Field) == 0 {
		return 0
	}
	var withID struct {
		ID int `json:"id"`
	}
	if err := json.Unmarshal(v.Field, &withID); err == nil && withID.ID != 0 {
		return withID.ID
	}
	var numeric int
	if err := json.Unmarshal(v.Field, &numeric); err == nil {
		return numeric
	}
	return 0
}

func (v customFieldValue) ValueString() string {
	if len(v.Value) > 0 && string(v.Value) != "null" {
		var str string
		if err := json.Unmarshal(v.Value, &str); err == nil {
			return strings.TrimSpace(str)
		}
		var boolean bool
		if err := json.Unmarshal(v.Value, &boolean); err == nil {
			if boolean {
				return "true"
			}
			return "false"
		}
		var list []string
		if err := json.Unmarshal(v.Value, &list); err == nil && len(list) > 0 {
			return strings.Join(list, ",")
		}
	}
	if v.StringValue != nil {
		return strings.TrimSpace(*v.StringValue)
	}
	if v.BoolValue != nil {
		if *v.BoolValue {
			return "true"
		}
		return "false"
	}
	if len(v.MultipleValue) > 0 {
		return strings.Join(v.MultipleValue, ",")
	}
	return ""
}

type agentDetails struct {
	AgentID      string             `json:"agent_id"`
	SiteID       int                `json:"site"`
	CustomFields []customFieldValue `json:"custom_fields"`
}

type siteDetails struct {
	ID           int                `json:"id"`
	ClientID     int                `json:"client"`
	CustomFields []customFieldValue `json:"custom_fields"`
}

type clientDetails struct {
	ID           int                `json:"id"`
	CustomFields []customFieldValue `json:"custom_fields"`
}

// FetchTrayData pulls Tactical RMM custom fields for tray customisation.
// Returns nil when integration is not configured.
func FetchTrayData(ctx context.Context, httpClient *http.Client, opts Options) (*TrayData, error) {
	baseURL := strings.TrimSpace(opts.BaseURL)
	apiKey := strings.TrimSpace(opts.APIKey)
	agentID := strings.TrimSpace(opts.AgentID)

	if baseURL == "" || apiKey == "" {
		logging.Debugf("skipping Tactical RMM lookup due to missing base URL or API key")
		return nil, nil
	}

	if agentID != "" && strings.EqualFold(agentID, apiKey) {
		logging.Debugf("ignoring Tactical RMM agent identifier that matches API key")
		agentID = ""
	}

	if httpClient == nil {
		httpClient = &http.Client{Timeout: 15 * time.Second}
	}

	if agentID == "" && opts.AgentPK > 0 {
		logging.Debugf("resolving Tactical RMM agent primary key %d", opts.AgentPK)
		resolved, err := resolveAgentIDFromPK(ctx, httpClient, baseURL, apiKey, opts.AgentPK)
		if err != nil {
			return nil, err
		}
		agentID = resolved
	}

	if agentID == "" {
		logging.Debugf("skipping Tactical RMM lookup due to missing agent identifier")
		return nil, nil
	}

	logging.Debugf("fetching Tactical RMM data for agent %s", logging.MaskIdentifier(agentID))

	defs, err := fetchCustomFieldDefinitions(ctx, httpClient, baseURL, apiKey)
	if err != nil {
		return nil, err
	}
	logging.Debugf("retrieved %d Tactical RMM custom field definitions", len(defs))

	agent, err := fetchAgent(ctx, httpClient, baseURL, apiKey, agentID)
	if err != nil {
		return nil, err
	}
	logging.Debugf("loaded Tactical RMM agent details for %s", logging.MaskIdentifier(agent.AgentID))

	warnings := newMultiError()

	siteID := opts.SiteID
	if siteID == 0 {
		siteID = agent.SiteID
	}

	var site *siteDetails
	if siteID > 0 {
		site, err = fetchSite(ctx, httpClient, baseURL, apiKey, siteID)
		if err != nil {
			if errors.Is(err, errNotFound) {
				warnings.add(fmt.Errorf("site %d not found", siteID))
				logging.Debugf("site %d not found when fetching Tactical RMM data", siteID)
			} else {
				return nil, err
			}
		}
		if site != nil {
			logging.Debugf("loaded Tactical RMM site %d", site.ID)
		}
	}

	clientID := opts.ClientID
	if clientID == 0 {
		if site != nil && site.ClientID > 0 {
			clientID = site.ClientID
		}
	}

	var client *clientDetails
	if clientID > 0 {
		client, err = fetchClient(ctx, httpClient, baseURL, apiKey, clientID)
		if err != nil {
			if errors.Is(err, errNotFound) {
				warnings.add(fmt.Errorf("client %d not found", clientID))
				logging.Debugf("client %d not found when fetching Tactical RMM data", clientID)
			} else {
				return nil, err
			}
		}
		if client != nil {
			logging.Debugf("loaded Tactical RMM client %d", client.ID)
		}
	}

	iconValue := firstNonEmpty(
		extractFieldValue(agent.CustomFields, findDefinition(defs, "agent", "TrayIcon")),
		extractFieldValue(siteFields(site), findDefinition(defs, "site", "TrayIcon")),
		extractFieldValue(clientFields(client), findDefinition(defs, "client", "TrayIcon")),
		defaultValue(findDefinition(defs, "agent", "TrayIcon")),
		defaultValue(findDefinition(defs, "site", "TrayIcon")),
		defaultValue(findDefinition(defs, "client", "TrayIcon")),
	)

	var iconData []byte
	if iconValue != "" {
		decoded, err := decodeBase64(iconValue)
		if err != nil {
			warnings.add(fmt.Errorf("decode tray icon: %w", err))
			logging.Debugf("failed to decode Tactical RMM icon payload: %v", err)
		} else {
			iconData = decoded
			logging.Debugf("decoded Tactical RMM icon payload (%d bytes)", len(iconData))
		}
	}

	menuValue := firstNonEmpty(
		extractFieldValue(agent.CustomFields, findDefinition(defs, "agent", "TrayMenu")),
		extractFieldValue(siteFields(site), findDefinition(defs, "site", "TrayMenu")),
		extractFieldValue(clientFields(client), findDefinition(defs, "client", "TrayMenu")),
		defaultValue(findDefinition(defs, "agent", "TrayMenu")),
		defaultValue(findDefinition(defs, "site", "TrayMenu")),
		defaultValue(findDefinition(defs, "client", "TrayMenu")),
	)

	var menuItems []config.MenuItem
	if strings.TrimSpace(menuValue) != "" {
		parsed, err := parseMenu(menuValue)
		if err != nil {
			warnings.add(fmt.Errorf("parse tray menu: %w", err))
			logging.Debugf("failed to parse Tactical RMM tray menu JSON: %v", err)
		} else {
			menuItems = parsed
			logging.Debugf("parsed %d Tactical RMM menu items", len(menuItems))
		}
	}

	if len(menuItems) == 0 && len(iconData) == 0 {
		if warnErr := warnings.err(); warnErr != nil {
			return nil, warnErr
		}
		return nil, nil
	}

	data := &TrayData{MenuItems: menuItems, Icon: iconData}
	return data, warnings.err()
}

func siteFields(site *siteDetails) []customFieldValue {
	if site == nil {
		return nil
	}
	return site.CustomFields
}

func clientFields(client *clientDetails) []customFieldValue {
	if client == nil {
		return nil
	}
	return client.CustomFields
}

func decodeBase64(value string) ([]byte, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil, nil
	}

	if idx := strings.Index(trimmed, ","); idx > -1 && strings.Contains(trimmed[:idx], "base64") {
		trimmed = trimmed[idx+1:]
	}
	trimmed = strings.TrimSpace(trimmed)
	if trimmed == "" {
		return nil, nil
	}

	data, err := base64.StdEncoding.DecodeString(trimmed)
	if err != nil {
		data, err = base64.RawStdEncoding.DecodeString(trimmed)
		if err != nil {
			return nil, err
		}
	}
	return data, nil
}

func parseMenu(value string) ([]config.MenuItem, error) {
	payload := strings.TrimSpace(value)
	if payload == "" {
		return nil, nil
	}

	var items []config.MenuItem
	if err := json.Unmarshal([]byte(payload), &items); err == nil {
		return items, nil
	}

	var wrapper struct {
		Items []config.MenuItem `json:"items"`
	}
	if err := json.Unmarshal([]byte(payload), &wrapper); err == nil && len(wrapper.Items) > 0 {
		return wrapper.Items, nil
	}

	return nil, fmt.Errorf("unsupported tray menu payload")
}

func fetchCustomFieldDefinitions(ctx context.Context, client *http.Client, baseURL, apiKey string) ([]customFieldDefinition, error) {
	endpoint, err := joinURL(baseURL, "/core/customfields/")
	if err != nil {
		return nil, err
	}

	var defs []customFieldDefinition
	if err := getJSON(ctx, client, endpoint, apiKey, &defs); err != nil {
		return nil, err
	}
	return defs, nil
}

func fetchAgent(ctx context.Context, httpClient *http.Client, baseURL, apiKey, agentID string) (*agentDetails, error) {
	endpoint, err := joinURL(baseURL, "/agents/"+url.PathEscape(agentID)+"/")
	if err != nil {
		return nil, err
	}
	var agent agentDetails
	if err := getJSON(ctx, httpClient, endpoint, apiKey, &agent); err != nil {
		return nil, err
	}
	return &agent, nil
}

func fetchSite(ctx context.Context, httpClient *http.Client, baseURL, apiKey string, id int) (*siteDetails, error) {
	endpoint, err := joinURL(baseURL, "/clients/sites/"+strconv.Itoa(id)+"/")
	if err != nil {
		return nil, err
	}
	var site siteDetails
	if err := getJSON(ctx, httpClient, endpoint, apiKey, &site); err != nil {
		return nil, err
	}
	return &site, nil
}

func fetchClient(ctx context.Context, httpClient *http.Client, baseURL, apiKey string, id int) (*clientDetails, error) {
	endpoint, err := joinURL(baseURL, "/clients/"+strconv.Itoa(id)+"/")
	if err != nil {
		return nil, err
	}
	var record clientDetails
	if err := getJSON(ctx, httpClient, endpoint, apiKey, &record); err != nil {
		return nil, err
	}
	return &record, nil
}

func resolveAgentIDFromPK(ctx context.Context, httpClient *http.Client, baseURL, apiKey string, pk int) (string, error) {
	if pk <= 0 {
		return "", errors.New("invalid agent primary key")
	}
	endpoint, err := joinURL(baseURL, "/agents/")
	if err != nil {
		return "", err
	}
	endpoint = endpoint + "?detail=1&page_size=2000"

	type agentSummary struct {
		ID      int    `json:"id"`
		AgentID string `json:"agent_id"`
	}

	var summaries []agentSummary
	if err := getJSON(ctx, httpClient, endpoint, apiKey, &summaries); err != nil {
		return "", err
	}

	for _, summary := range summaries {
		if summary.ID == pk {
			trimmed := strings.TrimSpace(summary.AgentID)
			if trimmed == "" {
				return "", fmt.Errorf("agent %d missing identifier", pk)
			}
			logging.Debugf("resolved Tactical RMM agent primary key %d to identifier %s", pk, logging.MaskIdentifier(trimmed))
			return trimmed, nil
		}
	}
	return "", errNotFound
}

func joinURL(base, path string) (string, error) {
	trimmed := strings.TrimSpace(base)
	if trimmed == "" {
		return "", errors.New("missing base URL")
	}
	trimmed = strings.TrimRight(trimmed, "/")
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return trimmed + path, nil
}

func getJSON(ctx context.Context, client *http.Client, endpoint, apiKey string, dest interface{}) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	req.Header.Set("X-API-KEY", apiKey)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return errNotFound
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("request failed (%d): %s", resp.StatusCode, strings.TrimSpace(string(snippet)))
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return err
	}
	if len(body) == 0 {
		return nil
	}

	// Some Tactical RMM endpoints wrap data in a "results" envelope.
	if err := json.Unmarshal(body, dest); err == nil {
		return nil
	}
	var wrapper struct {
		Results json.RawMessage `json:"results"`
	}
	if err := json.Unmarshal(body, &wrapper); err != nil {
		return err
	}
	if len(wrapper.Results) == 0 {
		return nil
	}
	return json.Unmarshal(wrapper.Results, dest)
}

func findDefinition(defs []customFieldDefinition, model, name string) *customFieldDefinition {
	for idx := range defs {
		if !strings.EqualFold(defs[idx].Model, model) {
			continue
		}
		if strings.EqualFold(defs[idx].Name, name) {
			return &defs[idx]
		}
	}
	return nil
}

func extractFieldValue(values []customFieldValue, def *customFieldDefinition) string {
	if def == nil {
		return ""
	}
	id := def.ID
	if id == 0 {
		return ""
	}
	for _, value := range values {
		if value.FieldID() == id {
			if v := value.ValueString(); v != "" {
				return v
			}
		}
	}
	return ""
}

func defaultValue(def *customFieldDefinition) string {
	if def == nil {
		return ""
	}
	return strings.TrimSpace(def.Default())
}

type multiError struct {
	parts []string
}

func newMultiError() *multiError {
	return &multiError{parts: make([]string, 0)}
}

func (m *multiError) add(err error) {
	if err == nil {
		return
	}
	m.parts = append(m.parts, err.Error())
}

func (m *multiError) err() error {
	if len(m.parts) == 0 {
		return nil
	}
	return errors.New(strings.Join(m.parts, "; "))
}
