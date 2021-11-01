package introspect

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/vasu1124/introspect/pkg/version"
)

var short bool

// versionCmd represents the server command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "states the exact version",

	Run: func(cmd *cobra.Command, args []string) {
		if !short {
			fmt.Printf("%s/%s/%s\n", version.Version, version.Commit, version.Branch)
		} else {
			fmt.Printf("%s\n", version.Version)
		}
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
	versionCmd.PersistentFlags().BoolVar(&short, "short", false, "short version")
}
