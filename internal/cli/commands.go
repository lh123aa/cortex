package cli

func init() {
	RootCmd.PersistentFlags().StringVarP(&cfgPath, "config", "c", "", "config file path")
	RootCmd.PersistentFlags().StringVarP(&logLevel, "log-level", "l", "", "log level (debug/info/warn/error)")

	RootCmd.AddCommand(NewIndexCmd())
	RootCmd.AddCommand(NewSearchCmd())
	RootCmd.AddCommand(NewContextCmd())
	RootCmd.AddCommand(NewMCPCmd())
	RootCmd.AddCommand(NewServeCmd())
	RootCmd.AddCommand(NewStatusCmd())
	RootCmd.AddCommand(NewDedupCmd())
	RootCmd.AddCommand(NewWatchCmd())
	RootCmd.AddCommand(NewUsageCmd())
	RootCmd.AddCommand(NewVersionCmd())
	RootCmd.AddCommand(NewSetupCmd())
}
