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
			fmt.Printf("%#v\n", version.Get())
		} else {
			fmt.Printf("%v\n", version.Get())
		}
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
	versionCmd.PersistentFlags().BoolVar(&short, "short", false, "short version")
}
