package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	ResourceDomains map[string]string `yaml:"resource_domains"`
}

var (
	mode    = flag.String("mode", "full", "Mode: schema-tests|full")
	resources = flag.String("resources", "all", "Comma-separated resource prefixes or 'all'")
	dryRun  = flag.Bool("dry-run", true, "Dry run (no file modifications)")
	rootDir = flag.String("root", ".", "Repository root")
	backup  = flag.Bool("backup", true, "Create backups when modifying")
)

func main() {
	flag.Parse()
	cfg, err := loadConfig(filepath.Join(*rootDir, "migration_map.yaml"))
	if err != nil { fatal(err) }

	selected := map[string]struct{}{}
	if *resources != "all" {
		for _, r := range strings.Split(*resources, ",") { selected[strings.TrimSpace(r)] = struct{}{} }
	}

	providerDir := filepath.Join(*rootDir, "pkg", "provider")
	matches, err := filepath.Glob(filepath.Join(providerDir, "*_resource.go"))
	if err != nil { fatal(err) }

	var plans []MigrationPlan
	for _, file := range matches {
		base := filepath.Base(file)
		prefix := strings.TrimSuffix(base, "_resource.go")
		if len(selected) > 0 { if _, ok := selected[prefix]; !ok { continue } }
		domain := cfg.ResourceDomains[prefix]
		if domain == "" { continue }
		plans = append(plans, buildPlan(prefix, domain, file, *mode))
	}

	if len(plans) == 0 { fmt.Println("No resources to migrate"); return }

	for _, p := range plans { fmt.Println(p.Summary()) }
	if *dryRun { fmt.Println("Dry run complete; no changes applied."); return }

	if *backup { createBackup(plans, *rootDir) }

	for _, p := range plans {
		if err := executePlan(p, *mode); err != nil { fatal(err) }
	}

	fmt.Println("Migration complete.")
}

type MigrationPlan struct {
	Prefix       string
	Domain       string
	ProviderFile string
	TargetDir    string
	TargetFile   string
	ShimFile     string
	Mode         string
}

func buildPlan(prefix, domain, providerFile, mode string) MigrationPlan {
	d := filepath.Join("pkg", "resources", domain)
	return MigrationPlan{
		Prefix:       prefix,
		Domain:       domain,
		ProviderFile: providerFile,
		TargetDir:    d,
		TargetFile:   filepath.Join(d, filepath.Base(providerFile)),
		ShimFile:     filepath.Join("pkg", "provider", filepath.Base(providerFile)),
		Mode:         mode,
	}
}

func (m MigrationPlan) Summary() string {
	return fmt.Sprintf("[%s] %s -> %s (mode=%s)", m.Prefix, m.ProviderFile, m.TargetFile, m.Mode)
}

func loadConfig(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil { return nil, err }
	var c Config
	if err := yaml.Unmarshal(b, &c); err != nil { return nil, err }
	return &c, nil
}

func createBackup(plans []MigrationPlan, root string) {
	stamp := time.Now().Format("20060102_150405")
	backupRoot := filepath.Join(root, "backup", "migration", stamp)
	_ = os.MkdirAll(backupRoot, 0o755)
	for _, p := range plans {
		data, err := os.ReadFile(p.ProviderFile)
		if err == nil {
			rel, _ := filepath.Rel(root, p.ProviderFile)
			path := filepath.Join(backupRoot, rel)
			_ = os.MkdirAll(filepath.Dir(path), 0o755)
			_ = os.WriteFile(path, data, 0o644)
		}
	}
	fmt.Println("Backups stored at", backupRoot)
}

func executePlan(p MigrationPlan, mode string) error {
	// Read provider file
	content, err := os.ReadFile(p.ProviderFile)
	if err != nil { return err }

	// If already shim (imports resources/<domain>) skip
	if bytes.Contains(content, []byte("legacy shim")) { return nil }

	if mode == "schema-tests" {
		// Only create schema test later (not implemented yet) and skip file move.
		return nil
	}

	// Ensure target dir
	if err := os.MkdirAll(p.TargetDir, 0o755); err != nil { return err }

	// Transform package name
	updated := transformPackage(string(content), p.Domain)

	// Write target file
	if err := os.WriteFile(p.TargetFile, []byte(updated), 0o644); err != nil { return err }

	// Replace provider file with shim delegating constructor
	shim := buildShim(p)
	if err := os.WriteFile(p.ProviderFile, []byte(shim), 0o644); err != nil { return err }

	return nil
}

var pkgRegex = regexp.MustCompile(`(?m)^package\s+provider`) // change to new package

func transformPackage(src, domain string) string {
	out := pkgRegex.ReplaceAllString(src, fmt.Sprintf("package %s", domain))
	// Replace helper func names where needed (traceAPICall -> common.TraceAPICall) naive approach
	out = strings.ReplaceAll(out, "traceAPICall(", "common.TraceAPICall(")
	out = strings.ReplaceAll(out, "TraceAPICall(", "common.TraceAPICall(")
	out = strings.ReplaceAll(out, "IsKnown(", "common.IsKnown(")
	out = strings.ReplaceAll(out, "waitForDatasetToBeReady", "common.WaitForDatasetToBeReady")
	out = strings.ReplaceAll(out, "addEntityToUseCase", "common.AddEntityToUseCase")
	out = strings.ReplaceAll(out, "updateUseCasesForEntity", "common.UpdateUseCasesForEntity")
	// Add imports if not present
	if !strings.Contains(out, "internal/common") {
		out = injectImport(out, "\t\"github.com/datarobot-community/terraform-provider-datarobot/internal/common\"")
	}
	return out
}

func injectImport(src, newImport string) string {
	lines := strings.Split(src, "\n")
	for i, l := range lines {
		if strings.HasPrefix(l, "import (") {
			// find closing )
			for j := i + 1; j < len(lines); j++ {
				if lines[j] == ")" { // insert before
					lines = append(lines[:j], append([]string{newImport}, lines[j:]...)...)
					return strings.Join(lines, "\n")
				}
			}
		}
	}
	// no import block found – create one after package line
	for i, l := range lines {
		if strings.HasPrefix(l, "package ") {
			block := []string{lines[i], "import (", newImport, ")"}
			block = append(block, lines[i+1:]...)
			return strings.Join(block, "\n")
		}
	}
	return src
}

func buildShim(p MigrationPlan) string {
	return fmt.Sprintf(`package provider

// legacy shim for %s_resource.go
// This file is auto-generated by tools/migrate.

import (
	res "%s"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func New%sResource() resource.Resource { return res.New%sResource() }
`, p.Prefix, moduleImportPath(p), camel(p.Prefix), camel(p.Prefix))
}

// Derive module path by reading go.mod (simple first line 'module X') once.
var cachedModule string

func moduleImportPath(p MigrationPlan) string { // accept plan for future, but only need root import
	if cachedModule != "" { return cachedModule + "/pkg/resources/" + p.Domain }
	data, err := os.ReadFile("go.mod")
	if err != nil { return "github.com/datarobot-community/terraform-provider-datarobot/pkg/resources/" + p.Domain }
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "module ") {
			cachedModule = strings.TrimSpace(strings.TrimPrefix(line, "module "))
			break
		}
	}
	if cachedModule == "" { cachedModule = "github.com/datarobot-community/terraform-provider-datarobot" }
	return cachedModule + "/pkg/resources/" + p.Domain
}

func camel(s string) string {
	parts := strings.Split(s, "_")
	for i, p := range parts {
		if len(p) == 0 { continue }
		parts[i] = strings.ToUpper(p[:1]) + p[1:]
	}
	return strings.Join(parts, "")
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "migration error:", err)
	os.Exit(1)
}

// Basic guard to ensure we don't overwrite already migrated domain file unexpectedly.
func fileExists(path string) bool { _, err := os.Stat(path); return err == nil }

// Placeholder for potential future validation of file content.
func validateTargetNotExists(p MigrationPlan) error {
	if fileExists(p.TargetFile) { return errors.New("target already exists: " + p.TargetFile) }
	return nil
}

// (Optional) walk function for future enhancements.
func walk(dir string, fn func(string, fs.FileInfo) error) error {
	return filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil { return err }
		info, _ := d.Info()
		if info == nil { return nil }
		return fn(path, info)
	})
}
