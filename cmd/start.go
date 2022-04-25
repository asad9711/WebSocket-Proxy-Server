/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"websocket_proxy_server/server_logic"

	"github.com/spf13/cobra"
)

// startCmd represents the start command
var startCmd = &cobra.Command{
	Use:   "start",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		portNumberRequested, _ := cmd.Flags().GetString("port")
		// fmt.Println("start called, with port   ", portNumberRequested)
		targetServer, _ := cmd.Flags().GetString("target")

		// set target server address
		server_logic.TargetURL = targetServer
		server_logic.SetupRouteAndStartServer(portNumberRequested)
	},
}

func init() {
	startCmd.PersistentFlags().String("port", "", "port number to start the proxy server at")
	startCmd.PersistentFlags().String("target", "", "target url")
	startCmd.MarkFlagRequired("target")

	rootCmd.AddCommand(startCmd)
	// fmt.Println("start called, with args  ")
	// server_logic.SetupRouteAndStartServer()
	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// startCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
