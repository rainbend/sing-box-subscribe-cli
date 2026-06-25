package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/rainbend/sing-box-subscribe-cli/internal/subconv"
	"github.com/rainbend/sing-box-subscribe-cli/internal/templates"
	"github.com/spf13/cobra"
)

const defaultTemplate = "sb-config-1.14.json"

var version = "dev"

func main() {
	if err := executeRootCommand(context.Background(), os.Args[1:], os.Stdout, os.Stderr); err != nil {
		os.Exit(1)
	}
}

func executeRootCommand(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	cmd := newRootCommand()
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetArgs(args)

	executed, err := cmd.ExecuteContextC(ctx)
	if err == nil {
		return nil
	}
	if executed == nil {
		executed = cmd
	}
	printHelpAndError(executed, stderr, err)
	return err
}

func printHelpAndError(cmd *cobra.Command, stderr io.Writer, err error) {
	cmd.SetOut(stderr)
	if helpErr := cmd.Help(); helpErr != nil {
		fmt.Fprintf(stderr, "help: %v\n", helpErr)
	}
	fmt.Fprintf(stderr, "\nerror: %v\n", err)
}

func newRootCommand() *cobra.Command {
	var opts subconv.Options
	var timeout time.Duration

	cmd := &cobra.Command{
		Use:           "sing-box-sub [subscription URL or file]",
		Short:         "Generate sing-box configs from subscriptions",
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				if opts.URL != "" {
					return fmt.Errorf("subscription source specified both as argument and --url")
				}
				opts.URL = args[0]
			}
			return runGenerate(cmd, opts, timeout)
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&opts.URL, "url", "", "subscription URL or local subscription file")
	flags.StringVar(&opts.Template, "template", defaultTemplate, "template name, template path, or URL")
	flags.StringVar(&opts.Output, "out", "config.json", "output config path, or - for stdout")
	flags.StringVar(&opts.Tag, "tag", "tag_1", "subscription group tag")
	flags.StringVar(&opts.UserAgent, "ua", "clashmeta", "User-Agent for subscription/template HTTP requests")
	flags.StringVar(&opts.Prefix, "prefix", "", "prefix added to generated outbound tags")
	flags.StringVar(&opts.ExcludeProtocol, "exclude-protocol", "ssr", "comma-separated protocols to skip")
	flags.StringVar(&opts.ExcludeNodeName, "exclude-node-name", "", "comma or pipe separated substrings to skip by tag")
	flags.BoolVar(&opts.OnlyNodes, "only-nodes", false, "write only generated outbounds instead of merging a template")
	flags.DurationVar(&timeout, "timeout", 60*time.Second, "HTTP request timeout")

	cmd.AddCommand(newListCommand(), newVersionCommand())
	return cmd
}

func runGenerate(cmd *cobra.Command, opts subconv.Options, timeout time.Duration) error {
	if opts.URL == "" {
		return fmt.Errorf("subscription URL or file is required")
	}
	if opts.Template == "" && !opts.OnlyNodes {
		return fmt.Errorf("--template is required unless --only-nodes is set")
	}
	if timeout <= 0 {
		return fmt.Errorf("--timeout must be positive")
	}
	opts.Timeout = timeout

	result, err := subconv.Generate(cmd.Context(), opts)
	if err != nil {
		return err
	}

	if err := writeJSON(opts.Output, result.Config); err != nil {
		return err
	}

	stderr := cmd.ErrOrStderr()
	fmt.Fprintf(stderr, "generated %d nodes", result.NodeCount)
	if len(result.Warnings) > 0 {
		fmt.Fprintf(stderr, " with %d warnings", len(result.Warnings))
	}
	fmt.Fprintln(stderr)
	for _, warning := range result.Warnings {
		fmt.Fprintf(stderr, "warning: %s\n", warning)
	}
	return nil
}

func writeJSON(path string, value any) error {
	var out *os.File
	if path == "-" {
		out = os.Stdout
	} else {
		file, err := os.Create(path)
		if err != nil {
			return fmt.Errorf("create output %q: %w", path, err)
		}
		defer file.Close()
		out = file
	}

	enc := json.NewEncoder(out)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(value); err != nil {
		return fmt.Errorf("write JSON: %w", err)
	}
	return nil
}

func newListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List available templates",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return listTemplates(cmd)
		},
	}
}

func listTemplates(cmd *cobra.Command) error {
	names, err := templates.List()
	if err != nil {
		return err
	}
	stdout := cmd.OutOrStdout()
	for _, name := range names {
		fmt.Fprintln(stdout, name)
	}
	return nil
}

func newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintln(cmd.OutOrStdout(), version)
			return nil
		},
	}
}
