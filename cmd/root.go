package cmd

import (
	"encoding/json"
	"fmt"
	"github.com/fotoetienne/gqai/mcp"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"os"

	"github.com/fotoetienne/gqai/graphql"
	"github.com/fotoetienne/gqai/tool"
	"github.com/spf13/cobra"
)

var config *graphql.GraphQLConfig
var configPath string

var rootCmd = &cobra.Command{
	Use:   "gqai",
	Short: "gqai - expose GraphQL operations as AI tools",
}

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run gqai as an MCP server in stdin/stdout mode",
	Run: func(cmd *cobra.Command, args []string) {
		mcp.RunMCPStdIO(config)
	},
}

var toolsCallCmd = &cobra.Command{
	Use:   "tools/call [toolName] [jsonInput]",
	Short: "Call a GraphQL operation as a tool",
	Args:  cobra.MinimumNArgs(1), // allow just the tool name
	Run: func(cmd *cobra.Command, args []string) {
		toolName := args[0]
		var input map[string]any
		if len(args) > 1 {
			if err := json.Unmarshal([]byte(args[1]), &input); err != nil {
				fmt.Println("Invalid JSON input:", err)
				os.Exit(1)
			}
		} else {
			input = map[string]any{} // default to empty input
		}

		var request = mcp.JSONRPCRequest{
			JSONRPC: "2.0",
			Method:  "tools/call",
			Params: map[string]any{
				"name":      toolName,
				"arguments": input,
			},
		}

		var resp = mcp.ToolsCall(request, config)

		var error = resp.Error
		if error != nil {
			fmt.Printf("Error: %s\n", error.Message)
			os.Exit(1)
		}

		var result = resp.Result
		out, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(out))
	},
}

var toolsListCmd = &cobra.Command{
	Use:   "tools/list",
	Short: "List available tools",
	Run: func(cmd *cobra.Command, args []string) {
		tools, err := tool.ToolsFromConfig(config)
		if err != nil {
			fmt.Println("Error loading tools:", err)
			os.Exit(1)
		}

		out, err := json.MarshalIndent(tools, "", "  ")
		if err != nil {
			fmt.Println("Error serializing tools:", err)
			os.Exit(1)
		}
		fmt.Println(string(out))
	},
}

var describeCmd = &cobra.Command{
	Use:   "describe [toolName]",
	Short: "Describe a tool and show its full schema",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		toolName := args[0]
		tools, err := tool.ToolsFromConfig(config)
		if err != nil {
			fmt.Println("Error loading tools:", err)
			os.Exit(1)
		}

		for _, t := range tools {
			if t.Name == toolName {
				out, _ := json.MarshalIndent(t, "", "  ")
				fmt.Println(string(out))
				return
			}
		}

		fmt.Printf("Tool %s not found\n", toolName)
	},
}

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Serve tools over HTTP",
	Run: func(cmd *cobra.Command, args []string) {
		r := mux.NewRouter()

		// List tools
		r.HandleFunc("/tools/list", listToolsHandler).Methods("GET")

		// Call a tool
		r.HandleFunc("/tools/call", callToolHandler).Methods("POST")

		// Tool specific handler
		r.HandleFunc("/tools/{toolName}", serveHandler).Methods("POST")

		fmt.Println("Serving on http://localhost:8080")
		log.Fatal(http.ListenAndServe("localhost:8080", r))
	},
}

func Execute() {
	rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", ".graphqlrc.yml", "Path to .graphqlrc.yml")

	cobra.OnInitialize(func() {
		var err error
		config, err = graphql.LoadGraphQLConfig(configPath)
		if err != nil {
			log.Fatalf("Error loading config: %v", err)
		}
	})

	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(toolsCallCmd)
	rootCmd.AddCommand(toolsListCmd)
	rootCmd.AddCommand(describeCmd)
	rootCmd.AddCommand(serveCmd)
	rootCmd.Execute()
}
