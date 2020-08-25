package cmd

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/podtserkovskiy/garnerd/app"
)

func Execute() {
	maxCount := 0
	rootCmd := &cobra.Command{
		Use:   "garnerd",
		Short: "Garnerd is a useful cache for docker",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.Start(maxCount, args[0])
		},
	}
	rootCmd.Flags().IntVar(&maxCount, "max-count", 10, "maximum images in the cache")

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
