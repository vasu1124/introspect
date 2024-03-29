package introspect

import (
	"github.com/spf13/cobra"
	"github.com/vasu1124/introspect/pkg/config"
	"github.com/vasu1124/introspect/pkg/logger"
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
		logger.Log.Info("[introspect] Version",
			"Version", version.Get().GitVersion,
			"Commit", version.Get().GitCommit,
			"TreeState", version.Get().GitTreeState,
			"BuildDate", version.Get().BuildDate,
			"GoVersion", version.Get().GoVersion,
			"Compiler", version.Get().Compiler,
			"Platform", version.Get().Platform,
		)

		stop := signal.Handler()
		srv := server.NewServer()
		srv.Run(stop)
	},
}

func init() {
	rootCmd.AddCommand(serverCmd)

	serverCmd.Flags().IntVarP(&config.Default.Port, "port", "p", config.Default.Port, "http port")
	serverCmd.Flags().IntVarP(&config.Default.SecurePort, "secure-port", "s", config.Default.SecurePort, "https port")
}
