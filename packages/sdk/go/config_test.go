// File generated from our OpenAPI spec by Stainless. See CONTRIBUTING.md for details.

package opencode_test

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/sst/opencode-sdk-go"
	"github.com/sst/opencode-sdk-go/internal/testutil"
	"github.com/sst/opencode-sdk-go/option"
	"github.com/sst/opencode-sdk-go/shared"
)

func TestConfigGetWithOptionalParams(t *testing.T) {
	t.Skip("Prism tests are disabled")
	baseURL := "http://localhost:4010"
	if envURL, ok := os.LookupEnv("TEST_API_BASE_URL"); ok {
		baseURL = envURL
	}
	if !testutil.CheckTestServer(t, baseURL) {
		return
	}
	client := opencode.NewClient(
		option.WithBaseURL(baseURL),
		option.WithAPIKey("My API Key"),
	)
	_, err := client.Config.Get(context.TODO(), opencode.ConfigGetParams{
		Directory: opencode.String("directory"),
	})
	if err != nil {
		var apierr *opencode.Error
		if errors.As(err, &apierr) {
			t.Log(string(apierr.DumpRequest(true)))
		}
		t.Fatalf("err should be nil: %s", err.Error())
	}
}

func TestConfigUpdateWithOptionalParams(t *testing.T) {
	t.Skip("Prism tests are disabled")
	baseURL := "http://localhost:4010"
	if envURL, ok := os.LookupEnv("TEST_API_BASE_URL"); ok {
		baseURL = envURL
	}
	if !testutil.CheckTestServer(t, baseURL) {
		return
	}
	client := opencode.NewClient(
		option.WithBaseURL(baseURL),
		option.WithAPIKey("My API Key"),
	)
	_, err := client.Config.Update(context.TODO(), opencode.ConfigUpdateParams{
		Directory: opencode.String("directory"),
		Configuration: opencode.ConfigurationParam{
			Schema: opencode.String("$schema"),
			Agent: opencode.ConfigurationAgentParam{
				Build: opencode.AgentConfigParam{
					Color:       opencode.String("#E1CB97"),
					Description: opencode.String("description"),
					Disable:     opencode.Bool(true),
					Mode:        opencode.AgentConfigModeSubagent,
					Model:       opencode.String("model"),
					Permission: opencode.AgentConfigPermissionParam{
						Bash: opencode.AgentConfigPermissionBashUnionParam{
							OfAgentConfigPermissionBashString: opencode.String("ask"),
						},
						DoomLoop:          "ask",
						Edit:              "ask",
						ExternalDirectory: "ask",
						Webfetch:          "ask",
					},
					Prompt:      opencode.String("prompt"),
					Temperature: opencode.Float(0),
					Tools: map[string]bool{
						"foo": true,
					},
					TopP: opencode.Float(0),
				},
				General: opencode.AgentConfigParam{
					Color:       opencode.String("#E1CB97"),
					Description: opencode.String("description"),
					Disable:     opencode.Bool(true),
					Mode:        opencode.AgentConfigModeSubagent,
					Model:       opencode.String("model"),
					Permission: opencode.AgentConfigPermissionParam{
						Bash: opencode.AgentConfigPermissionBashUnionParam{
							OfAgentConfigPermissionBashString: opencode.String("ask"),
						},
						DoomLoop:          "ask",
						Edit:              "ask",
						ExternalDirectory: "ask",
						Webfetch:          "ask",
					},
					Prompt:      opencode.String("prompt"),
					Temperature: opencode.Float(0),
					Tools: map[string]bool{
						"foo": true,
					},
					TopP: opencode.Float(0),
				},
				Plan: opencode.AgentConfigParam{
					Color:       opencode.String("#E1CB97"),
					Description: opencode.String("description"),
					Disable:     opencode.Bool(true),
					Mode:        opencode.AgentConfigModeSubagent,
					Model:       opencode.String("model"),
					Permission: opencode.AgentConfigPermissionParam{
						Bash: opencode.AgentConfigPermissionBashUnionParam{
							OfAgentConfigPermissionBashString: opencode.String("ask"),
						},
						DoomLoop:          "ask",
						Edit:              "ask",
						ExternalDirectory: "ask",
						Webfetch:          "ask",
					},
					Prompt:      opencode.String("prompt"),
					Temperature: opencode.Float(0),
					Tools: map[string]bool{
						"foo": true,
					},
					TopP: opencode.Float(0),
				},
			},
			Autoshare: opencode.Bool(true),
			Autoupdate: opencode.ConfigurationAutoupdateUnionParam{
				OfBool: opencode.Bool(true),
			},
			Command: map[string]opencode.ConfigurationCommandParam{
				"foo": {
					Template:    "template",
					Agent:       opencode.String("agent"),
					Description: opencode.String("description"),
					Model:       opencode.String("model"),
					Subtask:     opencode.Bool(true),
				},
			},
			DisabledProviders: []string{"string"},
			EnabledProviders:  []string{"string"},
			Enterprise: opencode.ConfigurationEnterpriseParam{
				URL: opencode.String("url"),
			},
			Experimental: opencode.ConfigurationExperimentalParam{
				BatchTool:           opencode.Bool(true),
				ChatMaxRetries:      opencode.Float(0),
				DisablePasteSummary: opencode.Bool(true),
				Hook: opencode.ConfigurationExperimentalHookParam{
					FileEdited: map[string][]opencode.ConfigurationExperimentalHookFileEditedParam{
						"foo": {{
							Command: []string{"string"},
							Environment: map[string]string{
								"foo": "string",
							},
						}},
					},
					SessionCompleted: []opencode.ConfigurationExperimentalHookSessionCompletedParam{{
						Command: []string{"string"},
						Environment: map[string]string{
							"foo": "string",
						},
					}},
				},
			},
			Formatter: opencode.ConfigurationFormatterUnionParam{
				OfBool: opencode.Bool(true),
			},
			Instructions: []string{"string"},
			Keybinds: opencode.ConfigurationKeybindsParam{
				AgentCycle:               opencode.String("agent_cycle"),
				AgentCycleReverse:        opencode.String("agent_cycle_reverse"),
				AgentList:                opencode.String("agent_list"),
				AppExit:                  opencode.String("app_exit"),
				CommandList:              opencode.String("command_list"),
				EditorOpen:               opencode.String("editor_open"),
				HistoryNext:              opencode.String("history_next"),
				HistoryPrevious:          opencode.String("history_previous"),
				InputClear:               opencode.String("input_clear"),
				InputForwardDelete:       opencode.String("input_forward_delete"),
				InputNewline:             opencode.String("input_newline"),
				InputPaste:               opencode.String("input_paste"),
				InputSubmit:              opencode.String("input_submit"),
				Leader:                   opencode.String("leader"),
				MessagesCopy:             opencode.String("messages_copy"),
				MessagesFirst:            opencode.String("messages_first"),
				MessagesHalfPageDown:     opencode.String("messages_half_page_down"),
				MessagesHalfPageUp:       opencode.String("messages_half_page_up"),
				MessagesLast:             opencode.String("messages_last"),
				MessagesPageDown:         opencode.String("messages_page_down"),
				MessagesPageUp:           opencode.String("messages_page_up"),
				MessagesRedo:             opencode.String("messages_redo"),
				MessagesToggleConceal:    opencode.String("messages_toggle_conceal"),
				MessagesUndo:             opencode.String("messages_undo"),
				ModelCycleRecent:         opencode.String("model_cycle_recent"),
				ModelCycleRecentReverse:  opencode.String("model_cycle_recent_reverse"),
				ModelList:                opencode.String("model_list"),
				SessionChildCycle:        opencode.String("session_child_cycle"),
				SessionChildCycleReverse: opencode.String("session_child_cycle_reverse"),
				SessionCompact:           opencode.String("session_compact"),
				SessionExport:            opencode.String("session_export"),
				SessionInterrupt:         opencode.String("session_interrupt"),
				SessionList:              opencode.String("session_list"),
				SessionNew:               opencode.String("session_new"),
				SessionShare:             opencode.String("session_share"),
				SessionTimeline:          opencode.String("session_timeline"),
				SessionUnshare:           opencode.String("session_unshare"),
				SidebarToggle:            opencode.String("sidebar_toggle"),
				StatusView:               opencode.String("status_view"),
				TerminalSuspend:          opencode.String("terminal_suspend"),
				ThemeList:                opencode.String("theme_list"),
			},
			Layout: opencode.ConfigurationLayoutAuto,
			Lsp: opencode.ConfigurationLspUnionParam{
				OfBool: opencode.Bool(true),
			},
			Mcp: map[string]opencode.ConfigurationMcpUnionParam{
				"foo": {
					OfMcpLocalConfig: &shared.McpLocalConfigParam{
						Command: []string{"string"},
						Enabled: opencode.Bool(true),
						Environment: map[string]string{
							"foo": "string",
						},
						Timeout: opencode.Int(1),
					},
				},
			},
			Mode: opencode.ConfigurationModeParam{
				Build: opencode.AgentConfigParam{
					Color:       opencode.String("#E1CB97"),
					Description: opencode.String("description"),
					Disable:     opencode.Bool(true),
					Mode:        opencode.AgentConfigModeSubagent,
					Model:       opencode.String("model"),
					Permission: opencode.AgentConfigPermissionParam{
						Bash: opencode.AgentConfigPermissionBashUnionParam{
							OfAgentConfigPermissionBashString: opencode.String("ask"),
						},
						DoomLoop:          "ask",
						Edit:              "ask",
						ExternalDirectory: "ask",
						Webfetch:          "ask",
					},
					Prompt:      opencode.String("prompt"),
					Temperature: opencode.Float(0),
					Tools: map[string]bool{
						"foo": true,
					},
					TopP: opencode.Float(0),
				},
				Plan: opencode.AgentConfigParam{
					Color:       opencode.String("#E1CB97"),
					Description: opencode.String("description"),
					Disable:     opencode.Bool(true),
					Mode:        opencode.AgentConfigModeSubagent,
					Model:       opencode.String("model"),
					Permission: opencode.AgentConfigPermissionParam{
						Bash: opencode.AgentConfigPermissionBashUnionParam{
							OfAgentConfigPermissionBashString: opencode.String("ask"),
						},
						DoomLoop:          "ask",
						Edit:              "ask",
						ExternalDirectory: "ask",
						Webfetch:          "ask",
					},
					Prompt:      opencode.String("prompt"),
					Temperature: opencode.Float(0),
					Tools: map[string]bool{
						"foo": true,
					},
					TopP: opencode.Float(0),
				},
			},
			Model: opencode.String("model"),
			Permission: opencode.ConfigurationPermissionParam{
				Bash: opencode.ConfigurationPermissionBashUnionParam{
					OfConfigurationPermissionBashString: opencode.String("ask"),
				},
				DoomLoop:          "ask",
				Edit:              "ask",
				ExternalDirectory: "ask",
				Webfetch:          "ask",
			},
			Plugin: []string{"string"},
			PromptVariables: map[string]string{
				"foo": "string",
			},
			Provider: map[string]opencode.ConfigurationProviderParam{
				"foo": {
					ID:        opencode.String("id"),
					API:       opencode.String("api"),
					Blacklist: []string{"string"},
					Env:       []string{"string"},
					Models: map[string]opencode.ConfigurationProviderModelParam{
						"foo": {
							ID:         opencode.String("id"),
							Attachment: opencode.Bool(true),
							Cost: opencode.ConfigurationProviderModelCostParam{
								Input:      0,
								Output:     0,
								CacheRead:  opencode.Float(0),
								CacheWrite: opencode.Float(0),
								ContextOver200k: opencode.ConfigurationProviderModelCostContextOver200kParam{
									Input:      0,
									Output:     0,
									CacheRead:  opencode.Float(0),
									CacheWrite: opencode.Float(0),
								},
							},
							Experimental: opencode.Bool(true),
							Headers: map[string]string{
								"foo": "string",
							},
							Limit: opencode.ConfigurationProviderModelLimitParam{
								Context: 0,
								Output:  0,
							},
							Modalities: opencode.ConfigurationProviderModelModalitiesParam{
								Input:  []string{"text"},
								Output: []string{"text"},
							},
							Name: opencode.String("name"),
							Options: map[string]any{
								"foo": "bar",
							},
							Provider: opencode.ConfigurationProviderModelProviderParam{
								Npm: "npm",
							},
							Reasoning:   opencode.Bool(true),
							ReleaseDate: opencode.String("release_date"),
							Status:      "alpha",
							Temperature: opencode.Bool(true),
							ToolCall:    opencode.Bool(true),
						},
					},
					Name: opencode.String("name"),
					Npm:  opencode.String("npm"),
					Options: opencode.ConfigurationProviderOptionsParam{
						APIKey:        opencode.String("apiKey"),
						BaseURL:       opencode.String("baseURL"),
						EnterpriseURL: opencode.String("enterpriseUrl"),
						Timeout: opencode.ConfigurationProviderOptionsTimeoutUnionParam{
							OfInt: opencode.Int(1),
						},
					},
					Whitelist: []string{"string"},
				},
			},
			Share:      opencode.ConfigurationShareManual,
			SmallModel: opencode.String("small_model"),
			Snapshot:   opencode.Bool(true),
			Theme:      opencode.String("theme"),
			Tools: map[string]bool{
				"foo": true,
			},
			Tui: opencode.ConfigurationTuiParam{
				ScrollAcceleration: opencode.ConfigurationTuiScrollAccelerationParam{
					Enabled: true,
				},
				ScrollSpeed: opencode.Float(0.001),
			},
			Username: opencode.String("username"),
			Watcher: opencode.ConfigurationWatcherParam{
				Ignore: []string{"string"},
			},
		},
	})
	if err != nil {
		var apierr *opencode.Error
		if errors.As(err, &apierr) {
			t.Log(string(apierr.DumpRequest(true)))
		}
		t.Fatalf("err should be nil: %s", err.Error())
	}
}

func TestConfigListProvidersWithOptionalParams(t *testing.T) {
	t.Skip("Prism tests are disabled")
	baseURL := "http://localhost:4010"
	if envURL, ok := os.LookupEnv("TEST_API_BASE_URL"); ok {
		baseURL = envURL
	}
	if !testutil.CheckTestServer(t, baseURL) {
		return
	}
	client := opencode.NewClient(
		option.WithBaseURL(baseURL),
		option.WithAPIKey("My API Key"),
	)
	_, err := client.Config.ListProviders(context.TODO(), opencode.ConfigListProvidersParams{
		Directory: opencode.String("directory"),
	})
	if err != nil {
		var apierr *opencode.Error
		if errors.As(err, &apierr) {
			t.Log(string(apierr.DumpRequest(true)))
		}
		t.Fatalf("err should be nil: %s", err.Error())
	}
}
