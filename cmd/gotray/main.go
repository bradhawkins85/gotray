package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/example/gotray/internal/config"
	"github.com/example/gotray/internal/menu"
)

func main() {
	log.SetFlags(0)
	secret := os.Getenv("GOTRAY_SECRET")
	if secret == "" {
		log.Fatal("GOTRAY_SECRET environment variable is required")
	}

	cfg, err := config.Load(secret)
	if err != nil {
		log.Fatalf("failed to load configuration: %v", err)
	}

	if len(os.Args) > 1 {
		if err := handleCLI(cfg, secret, os.Args[1:]); err != nil {
			log.Fatalf("%v", err)
		}
		return
	}

	if len(cfg.Items) == 0 {
		cfg.Items = menu.DefaultItems()
		if err := config.Save(cfg, secret); err != nil {
			log.Fatalf("failed to save default configuration: %v", err)
		}
	}

	ctx := context.Background()
	runner := menu.NewRunner(cfg)
	if err := runner.Start(ctx); err != nil {
		log.Fatalf("tray exited with error: %v", err)
	}
}

func handleCLI(cfg *config.Config, secret string, args []string) error {
	if len(args) == 0 {
		return errors.New("no command provided")
	}

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
	default:
		return fmt.Errorf("unknown command: %s", args[0])
	}
}

func normalizeCommand(arg string) string {
	trimmed := strings.TrimLeft(arg, "-/")
	return strings.ToLower(trimmed)
}

func handleAdd(cfg *config.Config, secret string, args []string) error {
	fs := newFlagSet("add")
	itemType := fs.String("type", string(config.MenuItemText), "menu item type: text, divider, command, url")
	label := fs.String("label", "", "display label")
	command := fs.String("command", "", "command or executable path")
	argList := fs.String("args", "", "comma-separated command arguments")
	workDir := fs.String("workdir", "", "working directory for command execution")
	url := fs.String("url", "", "target URL")
	description := fs.String("description", "", "tooltip description")

	if err := fs.Parse(args); err != nil {
		return err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	item := config.MenuItem{
		ID:          uuid.NewString(),
		Type:        config.MenuItemType(strings.ToLower(*itemType)),
		Label:       *label,
		Command:     *command,
		Arguments:   parseList(*argList),
		WorkingDir:  *workDir,
		URL:         *url,
		Description: *description,
		CreatedUTC:  now,
		UpdatedUTC:  now,
	}

	if err := validateItem(item); err != nil {
		return err
	}

	cfg.Items = append(cfg.Items, item)
	if err := config.Save(cfg, secret); err != nil {
		return err
	}

	fmt.Printf("Added menu item %s of type %s\n", item.ID, item.Type)
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

	if err := fs.Parse(args); err != nil {
		return err
	}

	if *id == "" {
		return errors.New("missing --id for update")
	}

	idx := -1
	for i, item := range cfg.Items {
		if item.ID == *id {
			idx = i
			break
		}
	}
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
	item.UpdatedUTC = time.Now().UTC().Format(time.RFC3339)

	if err := validateItem(item); err != nil {
		return err
	}

	cfg.Items[idx] = item
	if err := config.Save(cfg, secret); err != nil {
		return err
	}

	fmt.Printf("Updated menu item %s\n", item.ID)
	return nil
}

func handleDelete(cfg *config.Config, secret string, args []string) error {
	fs := newFlagSet("delete")
	id := fs.String("id", "", "identifier of the menu item to delete")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if *id == "" {
		return errors.New("missing --id for delete")
	}

	filtered := cfg.Items[:0]
	removed := false
	for _, item := range cfg.Items {
		if item.ID == *id {
			removed = true
			continue
		}
		filtered = append(filtered, item)
	}
	if !removed {
		return fmt.Errorf("item with id %s not found", *id)
	}

	cfg.Items = append([]config.MenuItem(nil), filtered...)
	if err := config.Save(cfg, secret); err != nil {
		return err
	}

	fmt.Printf("Deleted menu item %s\n", *id)
	return nil
}

func handleList(cfg *config.Config) error {
	if len(cfg.Items) == 0 {
		fmt.Println("No menu items configured")
		return nil
	}

	sort.Slice(cfg.Items, func(i, j int) bool {
		return cfg.Items[i].CreatedUTC < cfg.Items[j].CreatedUTC
	})

	fmt.Printf("%-38s %-8s %-20s %-20s\n", "ID", "Type", "Label", "Updated (UTC)")
	for _, item := range cfg.Items {
		fmt.Printf("%-38s %-8s %-20s %-20s\n", item.ID, item.Type, truncate(item.Label, 20), item.UpdatedUTC)
	}
	return nil
}

func validateItem(item config.MenuItem) error {
	switch item.Type {
	case config.MenuItemText:
		if item.Label == "" {
			return errors.New("text items require --label")
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
		// nothing required
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
