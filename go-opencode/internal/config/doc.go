// Package config provides configuration loading, merging, and path management for OpenCode.
//
// This package handles the complex configuration system that supports multiple sources
// and formats, with a hierarchical loading strategy that ensures proper precedence
// and compatibility with both TypeScript and Go implementations.
//
// # Configuration Loading
//
// The Load function implements a sophisticated configuration loading strategy that
// searches for and merges configuration from multiple sources in priority order:
//
//  1. Global config (~/.opencode/ - TypeScript compatible)
//  2. Global config (~/.config/opencode/ - XDG compatible)
//  3. Project configs discovered while walking up from the working directory
//     (opencode.json/opencode.jsonc and .opencode/opencode.json/opencode.jsonc)
//  4. OPENCODE_CONFIG file
//  5. OPENCODE_CONFIG_CONTENT inline JSON
//  6. Environment variables
//
// Configuration files are loaded in a specific order to ensure that more specific
// configurations override more general ones, while environment variables have the
// highest precedence.
//
// # Supported Formats
//
// The package supports both JSON and JSONC (JSON with Comments) formats:
//   - opencode.json - Standard JSON configuration
//   - opencode.jsonc - JSON with comments, processed using tidwall/jsonc
//
// # Variable Interpolation
//
// Configuration files support two types of variable interpolation:
//   - {env:VAR_NAME} - Expands to environment variable values
//   - {file:path} - Expands to file contents (properly escaped for JSON)
//
// File paths in {file:path} placeholders support:
//   - Absolute paths
//   - Relative paths (resolved relative to config file directory)
//   - Home directory expansion (~/)
//
// Example configuration with interpolation:
//
//	{
//	  "provider": {
//	    "anthropic": {
//	      "options": {
//	        "apiKey": "{env:ANTHROPIC_API_KEY}"
//	      }
//	    }
//	  },
//	  "instructions": [
//	    "{file:~/custom-instructions.txt}"
//	  ]
//	}
//
// # Configuration Merging
//
// When multiple configuration sources are found, they are merged using a deep merge
// strategy that:
//   - Overwrites scalar values (strings, booleans, numbers)
//   - Merges maps/objects by combining keys
//   - Appends to arrays/slices
//   - Preserves the last-loaded value for conflicts
//
// # Path Management
//
// The package provides XDG Base Directory Specification compliant path management
// through the Paths type:
//   - Data: ~/.local/share/opencode (XDG_DATA_HOME)
//   - Config: ~/.config/opencode (XDG_CONFIG_HOME)
//   - Cache: ~/.cache/opencode (XDG_CACHE_HOME)
//   - State: ~/.local/state/opencode (XDG_STATE_HOME)
//
// On Windows, these paths are adapted to use APPDATA as appropriate.
//
// # Environment Variable Overrides
//
// Several environment variables provide direct configuration overrides:
//   - OPENCODE_MODEL - Override the default model
//   - OPENCODE_SMALL_MODEL - Override the small model
//   - OPENCODE_PERMISSION - JSON string for permission configuration
//   - OPENCODE_CONFIG - Path to a specific config file
//   - OPENCODE_CONFIG_CONTENT - Inline JSON configuration
//   - OPENCODE_CONFIG_DIR - Override the config directory location
//
// # TypeScript Compatibility
//
// The configuration system maintains compatibility with the TypeScript implementation
// by supporting the ~/.opencode directory structure and TypeScript-style provider
// configuration with Options objects.
//
// # Usage Example
//
//	// Load configuration from the current directory
//	config, err := config.Load(".")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Get standard paths
//	paths := config.GetPaths()
//	err = paths.EnsurePaths() // Create directories if they don't exist
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Save configuration
//	err = config.Save(config, paths.GlobalConfigPath())
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// # Project Structure Discovery
//
// The configuration loader walks up the directory tree from the specified starting
// directory, stopping at either:
//   - A directory containing a .git folder (Git repository root)
//   - The filesystem root
//
// This ensures that project-specific configurations are properly discovered while
// respecting project boundaries.
package config