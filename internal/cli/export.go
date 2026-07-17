package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/jhayashi1/ascii-tui/internal/export"
	"github.com/jhayashi1/ascii-tui/internal/library"
	"github.com/jhayashi1/ascii-tui/internal/pathutil"
)

func init() {
	var output string

	exportCmd := &cobra.Command{
		Use:   "export <frames>",
		Short: "Export a rendered ASCII animation as a GIF file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			input := pathutil.ExpandTilde(args[0])
			anim, err := library.Load(input)
			if err != nil {
				return err
			}

			path := pathutil.ExpandTilde(output)
			if path == "" {
				base := strings.TrimSuffix(filepath.Base(input), filepath.Ext(input))
				path = base + ".gif"
			}
			f, err := os.Create(path)
			if err != nil {
				return fmt.Errorf("creating gif file: %w", err)
			}
			if err := export.GIF(f, anim); err != nil {
				f.Close()
				return err
			}
			if err := f.Close(); err != nil {
				return fmt.Errorf("writing gif file: %w", err)
			}

			pxW, pxH := export.PixelSize(anim)
			fmt.Fprintf(cmd.OutOrStdout(), "exported %d frames (%dx%d px) to %s\n",
				len(anim.Frames), pxW, pxH, path)
			return nil
		},
	}

	exportCmd.Flags().StringVarP(&output, "output", "o", "", "output gif file (default: <frames name>.gif)")
	rootCmd.AddCommand(exportCmd)
}
