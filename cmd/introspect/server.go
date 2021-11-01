package introspect

import (
	"log"

	"github.com/spf13/cobra"
	"github.com/vasu1124/introspect/pkg/config"
	"github.com/vasu1124/introspect/pkg/server"
	"github.com/vasu1124/introspect/pkg/signal"
	"github.com/vasu1124/introspect/pkg/version"
)

// serverCmd represents the server command
var serverCmd = &cobra.Command{
	Use:              "server",
	Short:            "(default) run introspect server",
	PersistentPreRun: bindFlags,
	Run: func(cmd *cobra.Command, args []string) {
		log.Printf("[introspect] Version = %s/%s/%s", version.Version, version.Commit, version.Branch)

		stop := signal.SignalHandler()
		srv := server.NewServer()
		srv.Run(stop)
	},
}

func init() {
	rootCmd.AddCommand(serverCmd)

	serverCmd.Flags().IntVarP(&config.Config.Port, "port", "p", config.Config.Port, "http port")
	serverCmd.Flags().IntVarP(&config.Config.SecurePort, "secure-port", "s", config.Config.SecurePort, "https port")
}
