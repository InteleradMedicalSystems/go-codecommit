package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var clobber bool

const (
	envKeyAwsProfile        = "AWS_PROFILE"
	envKeyAwsSDKLoadConfig  = "AWS_SDK_LOAD_CONFIG"
	envKeyCodeCommitRoleArn = "GO_CODECOMMIT_ROLE_ARN"
)

func setSDKLoadConfig() error {
	if _, isset := os.LookupEnv(envKeyAwsProfile); !isset {
		return nil
	}
	if _, isset := os.LookupEnv(envKeyAwsSDKLoadConfig); !isset {
		if err := os.Setenv(envKeyAwsSDKLoadConfig, "1"); err != nil {
			return err
		}
	}
	return nil
}

func main() {
	rootCmd := &cobra.Command{
		Use:   "codecommit",
		Short: "Tool for working with AWS' CodeCommit (Git) service",
	}

	// silence usage on Error
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		rootCmd.SilenceUsage = true
		return nil
	}

	if err := setSDKLoadConfig(); err != nil {
		fmt.Print(err)
		os.Exit(1)
	}

	rootCmd.AddCommand(newCredentialsCmd())
	rootCmd.AddCommand(newCredentialHelperCmd())
	rootCmd.AddCommand(newCloneCmd())
	rootCmd.AddCommand(newPullCmd())
	rootCmd.AddCommand(newPushCmd())
	rootCmd.AddCommand(newVersionCmd())

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
