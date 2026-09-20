package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/smford/tf-blast/pkg/config"
)

func newInitCmd() *cobra.Command {
	var (
		force    bool
		outPath  string
		toStdout bool
	)

	cmd := &cobra.Command{
		Use:   "init [path]",
		Short: "Initialize a starter .tf-blast.yaml policy configuration file",
		Long: `Initialize a starter .tf-blast.yaml policy configuration file.

Generates a fully commented policy configuration file with official JSON schema
validation, default risk severity classification rules, and CI gate thresholds.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if toStdout {
				_, err := fmt.Fprint(cmd.OutOrStdout(), config.DefaultTemplate)
				return err
			}

			target := ".tf-blast.yaml"
			if len(args) > 0 && args[0] != "" {
				target = args[0]
			} else if outPath != "" {
				target = outPath
			}

			target = filepath.Clean(target)

			// Check if file exists unless force is set
			if !force {
				if _, err := os.Stat(target); err == nil {
					return fmt.Errorf("configuration file %q already exists; use --force to overwrite", target)
				}
			}

			// Ensure parent directory exists if target specifies subdirectories
			parentDir := filepath.Dir(target)
			if parentDir != "" && parentDir != "." {
				if err := os.MkdirAll(parentDir, 0755); err != nil {
					return fmt.Errorf("failed to create directory %q: %w", parentDir, err)
				}
			}

			if err := os.WriteFile(target, []byte(config.DefaultTemplate), 0644); err != nil {
				return fmt.Errorf("failed to write configuration file %q: %w", target, err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Created %s with default policy configuration and IDE schema validation.\n", target)
			return nil
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "Overwrite existing configuration file without prompting")
	cmd.Flags().StringVarP(&outPath, "out", "o", "", "Target file path (defaults to .tf-blast.yaml)")
	cmd.Flags().BoolVar(&toStdout, "stdout", false, "Print template configuration directly to standard output")

	return cmd
}
