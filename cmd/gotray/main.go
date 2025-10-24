package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/example/gotray/internal/config"
	"github.com/example/gotray/internal/logging"
	"github.com/example/gotray/internal/menu"
	"github.com/example/gotray/internal/trmm"
)

func main() {
	log.SetFlags(0)

	args := os.Args[1:]
	var err error
	args, debug, offline, importTRMM, err := parseGlobalFlags(args)
	if err != nil {
		log.Fatalf("%v", err)
	}
	if debug {
		logging.EnableDebug()
	}

	if len(args) == 0 && importTRMM {
		secret := resolveSecret()
		if err := importFromTacticalRMM(secret); err != nil {
			log.Fatalf("failed to import Tactical RMM configuration: %v", err)
		}
		return
	}

	implicitMode := false
	if len(args) == 0 {
		implicitMode = true
		mode := strings.TrimSpace(os.Getenv("GOTRAY_RUN_MODE"))
		if mode == "" {
			mode = "run"
		}
		args = []string{mode}
	}

	switch normalizeCommand(args[0]) {
	case "run", "start":
		secret := resolveSecret()
		if importTRMM {
			if err := importFromTacticalRMM(secret); err != nil {
				log.Fatalf("failed to import Tactical RMM configuration: %v", err)
			}
		}
		if err := runStandalone(secret, offline); err != nil {
			log.Fatalf("tray execution failed: %v", err)
		}
		return
	}

	if implicitMode {
		log.Fatalf("unknown run mode %q; specify run, add, update, delete, list, move, export, or import", args[0])
	}

	secret := resolveSecret()
	if importTRMM {
		if err := importFromTacticalRMM(secret); err != nil {
			log.Fatalf("failed to import Tactical RMM configuration: %v", err)
		}
	}
	cfg, err := config.Load(secret)
	if err != nil {
		log.Fatalf("failed to load configuration: %v", err)
	}

	if err := handleCLI(cfg, secret, args); err != nil {
		log.Fatalf("%v", err)
	}
}

func runStandalone(secret string, offline bool) error {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	runner := menu.NewRunner(secret, offline)
	if err := runner.Start(ctx); err != nil {
		if errors.Is(err, context.Canceled) {
			return nil
		}
		return err
	}
	return nil
}

func resolveSecret() string {
	secret := strings.TrimSpace(config.CompiledSecret)
	if secret == "" {
		secret = strings.TrimSpace(os.Getenv("GOTRAY_SECRET"))
	}
	if secret == "" {
		log.Fatal("GOTRAY_SECRET secret is required; configure the GOTRAY_SECRET GitHub secret or set the GOTRAY_SECRET environment variable for local development")
	}
	return secret
}

func handleCLI(cfg *config.Config, secret string, args []string) error {
	if len(args) == 0 {
		return errors.New("no command provided")
	}

	menu.EnsureSequentialOrder(&cfg.Items)

	command := normalizeCommand(args[0])
	switch command {
	case "add":
		return handleAdd(cfg, secret, args[1:])
	case "update":
		return handleUpdate(cfg, secret, args[1:])
	case "delete":
		return handleDelete(cfg, secret, args[1:])
	case "list":
		return handleList(cfg)
	case "move":
		return handleMove(cfg, secret, args[1:])
	case "export":
		return handleExport(cfg)
	case "import":
		return handleImport(cfg, secret, args[1:])
	default:
		return fmt.Errorf("unknown command: %s", args[0])
	}
}

func normalizeCommand(arg string) string {
	trimmed := strings.TrimLeft(arg, "-/")
	return strings.ToLower(trimmed)
}

func parseGlobalFlags(args []string) ([]string, bool, bool, bool, error) {
	debugEnabled := false
	offlineEnabled := false
	importTRMM := false
	filtered := make([]string, 0, len(args))

	for _, arg := range args {
		normalized := strings.TrimSpace(arg)
		lower := strings.ToLower(strings.TrimLeft(normalized, "-/"))
		switch {
		case lower == "debug":
			debugEnabled = true
			continue
		case strings.HasPrefix(lower, "debug="):
			value := strings.TrimPrefix(lower, "debug=")
			parsed, err := strconv.ParseBool(value)
			if err != nil {
				return nil, false, fmt.Errorf("invalid value for --debug: %s", arg)
			}
			debugEnabled = parsed
			continue
		case lower == "offline":
			offlineEnabled = true
			continue
		case strings.HasPrefix(lower, "offline="):
			value := strings.TrimPrefix(lower, "offline=")
			parsed, err := strconv.ParseBool(value)
			if err != nil {
				return nil, false, false, false, fmt.Errorf("invalid value for --offline: %s", arg)
			}
			offlineEnabled = parsed
			continue
		case lower == "importtrmm":
			importTRMM = true
			continue
		case strings.HasPrefix(lower, "importtrmm="):
			value := strings.TrimPrefix(lower, "importtrmm=")
			parsed, err := strconv.ParseBool(value)
			if err != nil {
				return nil, false, false, false, fmt.Errorf("invalid value for --importtrmm: %s", arg)
			}
			importTRMM = parsed
			continue
		}
		filtered = append(filtered, arg)
	}

	return filtered, debugEnabled, offlineEnabled, importTRMM, nil
}

func importFromTacticalRMM(secret string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cfg, err := config.Load(secret)
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}

	options := trmm.DetectOptions()
	trayData, err := trmm.FetchTrayData(ctx, nil, options)
	if err != nil {
		return fmt.Errorf("fetch Tactical RMM tray data: %w", err)
	}
	if trayData == nil || len(trayData.MenuItems) == 0 {
		return errors.New("Tactical RMM did not provide any tray menu items to import")
	}

	items := make([]config.MenuItem, len(trayData.MenuItems))
	copy(items, trayData.MenuItems)

	now := time.Now().UTC().Format(time.RFC3339)
	seen := make(map[string]struct{}, len(items))
	for idx := range items {
		item := items[idx]
		if strings.TrimSpace(item.ID) == "" {
			item.ID = menu.GenerateID(items[:idx], item.ParentID, item.Type)
		}
		if _, exists := seen[item.ID]; exists {
			return fmt.Errorf("duplicate Tactical RMM menu item id %s", item.ID)
		}
		seen[item.ID] = struct{}{}

		if item.CreatedUTC == "" {
			item.CreatedUTC = now
		}
		if item.UpdatedUTC == "" {
			item.UpdatedUTC = item.CreatedUTC
		}

		if err := validateItem(item); err != nil {
			return fmt.Errorf("item %s invalid: %w", item.ID, err)
		}

		items[idx] = item
	}

	for _, item := range items {
		if err := menu.ValidateParent(items, item); err != nil {
			return fmt.Errorf("item %s invalid parent: %w", item.ID, err)
		}
	}

	menu.EnsureSequentialOrder(&items)

	cfg.Items = items
	if err := config.Save(cfg, secret); err != nil {
		return fmt.Errorf("save configuration: %w", err)
	}

	fmt.Printf("Imported %d Tactical RMM menu items into local configuration\n", len(items))
	return nil
}

func handleAdd(cfg *config.Config, secret string, args []string) error {
	fs := newFlagSet("add")
	itemType := fs.String("type", string(config.MenuItemText), "menu item type: text, divider, command, url, menu, quit")
	label := fs.String("label", "", "display label")
	command := fs.String("command", "", "command or executable path")
	argList := fs.String("args", "", "comma-separated command arguments")
	workDir := fs.String("workdir", "", "working directory for command execution")
	url := fs.String("url", "", "target URL")
	description := fs.String("description", "", "tooltip description")
	position := fs.Int("position", 0, "1-based position where the item should be inserted; defaults to the end")
	parent := fs.String("parent", "", "parent menu id for nested items")

	if err := fs.Parse(args); err != nil {
		return err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	normalizedType := config.MenuItemType(strings.ToLower(*itemType))
	parentID := strings.TrimSpace(*parent)
	item := config.MenuItem{
		ID:          menu.GenerateID(cfg.Items, parentID, normalizedType),
		Type:        normalizedType,
		Label:       *label,
		Command:     *command,
		Arguments:   parseList(*argList),
		WorkingDir:  *workDir,
		URL:         *url,
		Description: *description,
		ParentID:    parentID,
		CreatedUTC:  now,
		UpdatedUTC:  now,
	}

	if err := validateItem(item); err != nil {
		return err
	}

	if err := menu.ValidateParent(cfg.Items, item); err != nil {
		return err
	}

	idx := len(cfg.Items)
	if *position > 0 {
		idx = *position - 1
		if idx < 0 {
			idx = 0
		}
		if idx > len(cfg.Items) {
			idx = len(cfg.Items)
		}
	}

	cfg.Items = menu.InsertItem(cfg.Items, idx, item)
	menu.EnsureSequentialOrder(&cfg.Items)
	if err := config.Save(cfg, secret); err != nil {
		return err
	}

	fmt.Printf("Added menu item %s of type %s at position %d\n", item.ID, item.Type, idx+1)
	return nil
}

func handleUpdate(cfg *config.Config, secret string, args []string) error {
	fs := newFlagSet("update")
	id := fs.String("id", "", "identifier of the menu item to update")
	itemType := fs.String("type", "", "new item type")
	label := fs.String("label", "", "display label")
	command := fs.String("command", "", "command or executable path")
	argList := fs.String("args", "", "comma-separated command arguments")
	workDir := fs.String("workdir", "", "working directory")
	url := fs.String("url", "", "target URL")
	description := fs.String("description", "", "tooltip description")
	parent := fs.String("parent", "__unchanged__", "parent menu id (empty string for top level)")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if *id == "" {
		return errors.New("missing --id for update")
	}

	idx := findItemIndexByID(cfg.Items, *id)
	if idx == -1 {
		return fmt.Errorf("item with id %s not found", *id)
	}

	item := cfg.Items[idx]
	if *itemType != "" {
		item.Type = config.MenuItemType(strings.ToLower(*itemType))
	}
	if *label != "" {
		item.Label = *label
	}
	if *command != "" || (*itemType != "" && item.Type != config.MenuItemCommand) {
		item.Command = *command
	}
	if *argList != "" || (*itemType != "" && item.Type != config.MenuItemCommand) {
		item.Arguments = parseList(*argList)
	}
	if *workDir != "" || (*itemType != "" && item.Type != config.MenuItemCommand) {
		item.WorkingDir = *workDir
	}
	if *url != "" || (*itemType != "" && item.Type != config.MenuItemURL) {
		item.URL = *url
	}
	if *description != "" {
		item.Description = *description
	}
	if parent != nil && *parent != "__unchanged__" {
		item.ParentID = strings.TrimSpace(*parent)
	}
	item.UpdatedUTC = time.Now().UTC().Format(time.RFC3339)

	if err := validateItem(item); err != nil {
		return err
	}

	if err := menu.ValidateParent(cfg.Items, item); err != nil {
		return err
	}

	cfg.Items[idx] = item
	menu.EnsureSequentialOrder(&cfg.Items)
	if err := config.Save(cfg, secret); err != nil {
		return err
	}

	fmt.Printf("Updated menu item %s\n", item.ID)
	return nil
}

func handleDelete(cfg *config.Config, secret string, args []string) error {
	fs := newFlagSet("delete")
	id := fs.String("id", "", "identifier of the menu item to delete")
	label := fs.String("label", "", "label of the menu item to delete")
	deleteAll := fs.Bool("all", false, "remove all menu items")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if *deleteAll {
		if *id != "" || *label != "" {
			return errors.New("--all cannot be combined with --id or --label")
		}

		count := len(cfg.Items)
		if count == 0 {
			fmt.Println("No menu items to delete")
			return nil
		}

		cfg.Items = nil
		if err := config.Save(cfg, secret); err != nil {
			return err
		}

		fmt.Printf("Deleted all %d menu items\n", count)
		return nil
	}

	if *id == "" && *label == "" {
		return errors.New("specify --id or --label for delete")
	}

	descriptor := ""
	idx := -1
	if *id != "" {
		idx = findItemIndexByID(cfg.Items, *id)
		descriptor = fmt.Sprintf("id %s", *id)
	}
	if idx == -1 && *label != "" {
		idx = findItemIndexByLabel(cfg.Items, *label)
		descriptor = fmt.Sprintf("label %q", *label)
	}
	if idx == -1 {
		return fmt.Errorf("item with %s not found", descriptor)
	}

	removed := cfg.Items[idx]
	cfg.Items = menu.RemoveIndex(cfg.Items, idx)
	menu.EnsureSequentialOrder(&cfg.Items)
	if err := config.Save(cfg, secret); err != nil {
		return err
	}

	if *id != "" {
		fmt.Printf("Deleted menu item %s\n", removed.ID)
	} else {
		fmt.Printf("Deleted menu item %s with label %q\n", removed.ID, removed.Label)
	}
	return nil
}

func handleMove(cfg *config.Config, secret string, args []string) error {
	fs := newFlagSet("move")
	id := fs.String("id", "", "identifier of the menu item to move")
	label := fs.String("label", "", "label of the menu item to move")
	position := fs.Int("position", 0, "1-based position to move the item to")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if *id == "" && *label == "" {
		return errors.New("specify --id or --label for move")
	}
	if *position <= 0 {
		return errors.New("--position must be greater than zero")
	}

	descriptor := ""
	idx := -1
	if *id != "" {
		idx = findItemIndexByID(cfg.Items, *id)
		descriptor = fmt.Sprintf("id %s", *id)
	}
	if idx == -1 && *label != "" {
		idx = findItemIndexByLabel(cfg.Items, *label)
		descriptor = fmt.Sprintf("label %q", *label)
	}
	if idx == -1 {
		return fmt.Errorf("item with %s not found", descriptor)
	}

	item := cfg.Items[idx]
	item.UpdatedUTC = time.Now().UTC().Format(time.RFC3339)

	cfg.Items = menu.RemoveIndex(cfg.Items, idx)

	target := *position - 1
	if target < 0 {
		target = 0
	}
	if target > len(cfg.Items) {
		target = len(cfg.Items)
	}

	cfg.Items = menu.InsertItem(cfg.Items, target, item)
	menu.EnsureSequentialOrder(&cfg.Items)

	if err := config.Save(cfg, secret); err != nil {
		return err
	}

	fmt.Printf("Moved menu item %s to position %d\n", item.ID, target+1)
	return nil
}

func handleList(cfg *config.Config) error {
	if len(cfg.Items) == 0 {
		fmt.Println("No menu items configured")
		return nil
	}

	menu.EnsureSequentialOrder(&cfg.Items)

	fmt.Printf("%-5s %-38s %-8s %-12s %-20s %-20s\n", "Pos", "ID", "Type", "Parent", "Label", "Updated (UTC)")
	for idx, item := range cfg.Items {
		fmt.Printf("%-5d %-38s %-8s %-12s %-20s %-20s\n", idx+1, item.ID, item.Type, truncate(item.ParentID, 12), truncate(item.Label, 20), item.UpdatedUTC)
	}
	return nil
}

func handleExport(cfg *config.Config) error {
	menu.EnsureSequentialOrder(&cfg.Items)

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal configuration: %w", err)
	}

	encoded := base64.StdEncoding.EncodeToString(data)
	fmt.Println(encoded)
	return nil
}

func handleImport(cfg *config.Config, secret string, args []string) error {
	fs := newFlagSet("import")
	dataFlag := fs.String("data", "", "base64-encoded configuration payload")
	fileFlag := fs.String("file", "", "path to a file containing the base64 payload")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if *dataFlag == "" && *fileFlag == "" {
		return errors.New("provide --data or --file with the configuration payload")
	}
	if *dataFlag != "" && *fileFlag != "" {
		return errors.New("--data cannot be combined with --file")
	}

	payload := strings.TrimSpace(*dataFlag)
	if *fileFlag != "" {
		content, err := os.ReadFile(*fileFlag)
		if err != nil {
			return fmt.Errorf("read payload file: %w", err)
		}
		payload = strings.TrimSpace(string(content))
	}

	if payload == "" {
		return errors.New("configuration payload is empty")
	}

	decoded, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		return fmt.Errorf("decode payload: %w", err)
	}

	var imported config.Config
	if err := json.Unmarshal(decoded, &imported); err != nil {
		return fmt.Errorf("parse configuration: %w", err)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	seen := make(map[string]struct{})
	for idx := range imported.Items {
		item := imported.Items[idx]
		if item.ID == "" {
			return fmt.Errorf("imported item at position %d is missing an id", idx+1)
		}
		if _, exists := seen[item.ID]; exists {
			return fmt.Errorf("duplicate menu item id %s in imported configuration", item.ID)
		}
		seen[item.ID] = struct{}{}

		if item.CreatedUTC == "" {
			item.CreatedUTC = now
		}
		if item.UpdatedUTC == "" {
			item.UpdatedUTC = item.CreatedUTC
		}

		if err := validateItem(item); err != nil {
			return fmt.Errorf("item %s invalid: %w", item.ID, err)
		}

		imported.Items[idx] = item
	}

	for _, item := range imported.Items {
		if err := menu.ValidateParent(imported.Items, item); err != nil {
			return fmt.Errorf("item %s invalid parent: %w", item.ID, err)
		}
	}

	menu.EnsureSequentialOrder(&imported.Items)
	cfg.Items = imported.Items
	if err := config.Save(cfg, secret); err != nil {
		return err
	}

	fmt.Printf("Imported %d menu items\n", len(cfg.Items))
	return nil
}

func validateItem(item config.MenuItem) error {
	switch item.Type {
	case config.MenuItemText:
		if item.Label == "" {
			return errors.New("text items require --label")
		}
	case config.MenuItemMenu:
		if item.Label == "" {
			return errors.New("menu items require --label")
		}
		if item.ParentID == "" {
			return errors.New("menu items require --parent")
		}
	case config.MenuItemCommand:
		if item.Label == "" {
			return errors.New("command items require --label")
		}
		if item.Command == "" {
			return errors.New("command items require --command")
		}
	case config.MenuItemURL:
		if item.Label == "" {
			return errors.New("URL items require --label")
		}
		if item.URL == "" {
			return errors.New("URL items require --url")
		}
	case config.MenuItemDivider:
		if item.Label == "" {
			return errors.New("divider items require --label")
		}
	case config.MenuItemQuit:
		if item.Label == "" {
			return errors.New("quit items require --label")
		}
	default:
		return fmt.Errorf("unsupported menu type: %s", item.Type)
	}
	return nil
}

func parseList(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func truncate(value string, max int) string {
	if len(value) <= max {
		return value
	}
	if max <= 3 {
		return value[:max]
	}
	return value[:max-3] + "..."
}

func newFlagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	return fs
}

func findItemIndexByID(items []config.MenuItem, id string) int {
	for i := range items {
		if items[i].ID == id {
			return i
		}
	}
	return -1
}

func findItemIndexByLabel(items []config.MenuItem, label string) int {
	for i := range items {
		if strings.EqualFold(items[i].Label, label) {
			return i
		}
	}
	return -1
}
