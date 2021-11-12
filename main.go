package main

import (
	"fmt"
	"github.com/spf13/cobra"
	"os"
	"strings"
)

var rootCommand = &cobra.Command{}

var httpFlags struct {
	Credential string
}

var downloadFlags struct {
	OutputDir string
}

func parseCredential(value string) (credential *Credential, err error) {
	if len(value) == 0 {
		return
	}

	parts := strings.SplitN(value, ":", 2)

	if len(parts) == 1 {
		credential = &Credential{UserName: parts[0]}
		return
	}

	credential = &Credential{UserName: parts[0], Password: parts[1]}
	return
}

func newClient() (client *WebIndexClient, err error) {
	credential, err := parseCredential(httpFlags.Credential)

	if err != nil {
		return
	}

	client = &WebIndexClient{httpClient: makeHttpClient(), credential: credential}
	return
}

var listCommand = &cobra.Command{
	Use:  "list url",
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		url := args[0]
		client, err := newClient()

		if err != nil {
			return
		}

		return client.PrintEntries(url)
	},
}

var downloadCommand = &cobra.Command{
	Use:  "download url",
	Aliases: []string{"dl"},
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		url := args[0]
		client, err := newClient()

		if err != nil {
			return
		}

		return client.DownloadEntries(url, downloadFlags.OutputDir)
	},
}

func init() {
	listCommand.Flags().StringVarP(&httpFlags.Credential, "auth", "a", "", "Specify user and password of basic authentication")
	rootCommand.AddCommand(listCommand)

	downloadCommand.Flags().StringVarP(&httpFlags.Credential, "auth", "a", "", "Specify user and password of basic authentication")
	downloadCommand.Flags().StringVarP(&downloadFlags.OutputDir, "output-dir", "o", "", "Specify output directory")
	rootCommand.AddCommand(downloadCommand)
}

func main() {
	if err := rootCommand.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
