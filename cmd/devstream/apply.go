package main

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/devstream-io/devstream/internal/pkg/pluginengine"
	"github.com/devstream-io/devstream/pkg/util/log"
)

var applyCMD = &cobra.Command{
	Use:   "apply",
	Short: "Create or update DevOps tools according to DevStream configuration file",
	Long: `Create or update DevOps tools according to DevStream configuration file. 
DevStream will generate and execute a new plan based on the config file and the state file by default.`,
	Run: applyCMDFunc,
}

func applyCMDFunc(cmd *cobra.Command, args []string) {
	log.Info("Apply started.")
	if err := pluginengine.Apply(configFile, varFile, continueDirectly); err != nil {
		log.Errorf("Apply failed => %s.", err)
		os.Exit(1)
	}
	log.Success("Apply finished.")
}
