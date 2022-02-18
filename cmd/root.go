package cmd

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootCmd = &cobra.Command{
	Use:           "gitlab-registry-cleanup",
	Short:         "A tool for cleaning up gitlab registries",
	SilenceErrors: true,
	SilenceUsage:  true,
}

// Execute executes the root command.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(
		initConfig,
		initLogging,
	)

	rootCmd.PersistentFlags().String("config", "config.yml", "config file")
	rootCmd.PersistentFlags().Bool("debug", false, "specifies logging level should be set to debug")
	viper.BindPFlag("debug", rootCmd.PersistentFlags().Lookup("debug"))

	rootCmd.AddCommand(ExecuteCmd())
}

func initConfig() {
	config, _ := rootCmd.Flags().GetString("config")
	if config != "" {
		viper.SetConfigFile(config)
	} else {
		viper.AddConfigPath(".")
		viper.SetConfigName("config")
	}

	viper.AutomaticEnv()

	err := viper.ReadInConfig()
	if err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			log.Fatalf("Failed to load/parse config file: %s", err)
		}
		return
	}
	log.Infof("Using config file: %s", viper.ConfigFileUsed())
}

func initLogging() {
	if viper.GetBool("debug") {
		log.SetLevel(log.TraceLevel)
	}
}
