package main

import (
    "encoding/json"
    "errors"
    "os"
    "path/filepath"

    "github.com/spf13/pflag"
)

// CliOptions mirrors the TypeScript simple CLI options for parity.
type CliOptions struct {
    URL       string `json:"url"`
    APIKey    string `json:"apiKey"`
    Model     string `json:"model"`
    Provider  string `json:"provider"`
    Agent     string `json:"agent"`
    Session   string `json:"session"`
    Directory string `json:"directory"`
    Quiet     bool   `json:"quiet"`
    Verbose   bool   `json:"verbose"`
    JSON      bool   `json:"json"`
    NoColor   bool   `json:"noColor"`
    Trace     bool   `json:"trace"`
}

// ResolvedConfig contains fully merged configuration including derived paths.
type ResolvedConfig struct {
    CliOptions
    SessionFile string
}

func parseFlags(args []string) (*CliOptions, error) {
    flags := pflag.NewFlagSet("simple-go-cli", pflag.ContinueOnError)
    flags.String("url", "", "OpenCode server URL (or OPENCODE_SERVER_URL)")
    flags.String("api-key", "", "API key for authorization (or OPENCODE_API_KEY)")
    flags.String("model", "", "Model ID")
    flags.String("provider", "", "Provider ID")
    flags.String("agent", "", "Agent ID")
    flags.String("session", "", "Session ID")
    flags.String("directory", "", "Directory path to send to server")
    flags.Bool("quiet", false, "Silence banner and helper output")
    flags.Bool("verbose", false, "Enable verbose tracing to stderr")
    flags.Bool("json", false, "Emit JSON output for all events")
    flags.Bool("no-color", false, "Disable ANSI colors")
    flags.Bool("trace", false, "Enable trace logging from renderer")

    if err := flags.Parse(args); err != nil {
        return nil, err
    }

    opts := &CliOptions{}
    flags.VisitAll(func(f *pflag.Flag) {
        switch f.Name {
        case "url":
            opts.URL, _ = flags.GetString(f.Name)
        case "api-key":
            opts.APIKey, _ = flags.GetString(f.Name)
        case "model":
            opts.Model, _ = flags.GetString(f.Name)
        case "provider":
            opts.Provider, _ = flags.GetString(f.Name)
        case "agent":
            opts.Agent, _ = flags.GetString(f.Name)
        case "session":
            opts.Session, _ = flags.GetString(f.Name)
        case "directory":
            opts.Directory, _ = flags.GetString(f.Name)
        case "quiet":
            opts.Quiet, _ = flags.GetBool(f.Name)
        case "verbose":
            opts.Verbose, _ = flags.GetBool(f.Name)
        case "json":
            opts.JSON, _ = flags.GetBool(f.Name)
        case "no-color":
            opts.NoColor, _ = flags.GetBool(f.Name)
        case "trace":
            opts.Trace, _ = flags.GetBool(f.Name)
        }
    })

    return opts, nil
}

func loadConfigFile(cwd string) (*CliOptions, error) {
    homeDir, err := os.UserHomeDir()
    if err != nil {
        return nil, err
    }
    homeCfg := filepath.Join(homeDir, ".opencode", "simple-cli.json")
    projectCfg := filepath.Join(cwd, ".opencode", "simple-cli.json")

    merged := &CliOptions{}
    for _, path := range []string{homeCfg, projectCfg} {
        data, err := os.ReadFile(path)
        if err != nil {
            continue
        }
        var next CliOptions
        if err := json.Unmarshal(data, &next); err != nil {
            continue
        }
        applyOptions(merged, &next)
    }
    return merged, nil
}

func applyOptions(dst *CliOptions, src *CliOptions) {
    if src.URL != "" {
        dst.URL = src.URL
    }
    if src.APIKey != "" {
        dst.APIKey = src.APIKey
    }
    if src.Model != "" {
        dst.Model = src.Model
    }
    if src.Provider != "" {
        dst.Provider = src.Provider
    }
    if src.Agent != "" {
        dst.Agent = src.Agent
    }
    if src.Session != "" {
        dst.Session = src.Session
    }
    if src.Directory != "" {
        dst.Directory = src.Directory
    }
    if src.Quiet {
        dst.Quiet = true
    }
    if src.Verbose {
        dst.Verbose = true
    }
    if src.JSON {
        dst.JSON = true
    }
    if src.NoColor {
        dst.NoColor = true
    }
    if src.Trace {
        dst.Trace = true
    }
}

func resolveConfig(args []string) (ResolvedConfig, error) {
    cwd, err := os.Getwd()
    if err != nil {
        return ResolvedConfig{}, err
    }

    cliOpts, err := parseFlags(args)
    if err != nil {
        return ResolvedConfig{}, err
    }

    fileOpts, _ := loadConfigFile(cwd)
    merged := &CliOptions{}
    if fileOpts != nil {
        applyOptions(merged, fileOpts)
    }
    applyOptions(merged, cliOpts)

    if merged.URL == "" {
        merged.URL = os.Getenv("OPENCODE_SERVER_URL")
    }
    if merged.URL == "" {
        return ResolvedConfig{}, errors.New("missing server URL: pass --url or set OPENCODE_SERVER_URL")
    }

    if merged.APIKey == "" {
        merged.APIKey = os.Getenv("OPENCODE_API_KEY")
    }

    home, _ := os.UserHomeDir()
    sessionFile := filepath.Join(home, ".opencode", "simple-cli-state.json")

    return ResolvedConfig{CliOptions: *merged, SessionFile: sessionFile}, nil
}
