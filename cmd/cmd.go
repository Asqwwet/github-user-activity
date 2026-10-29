package cmd

import (
	"github_activity_project/activities"

	"github.com/spf13/cobra"
)

func NewRootCmd() *cobra.Command {
	return &cobra.Command{
		Use:           "github-activity <username>",
		Short:         "Displays recent activity of a user in github",
		SilenceErrors: true,
		SilenceUsage:  true,
		Args:          cobra.ExactArgs(1),
		/*
			so in here args is taking only one argument if there is no argument or more than 1 argument then it will produce error
		*/

		RunE: func(cmd *cobra.Command, args []string) error {
			return RunDisplayActivityCmd(args[0])
		},
	}
}

func RunDisplayActivityCmd(username string) error {
	userActivities, err := activities.GetUserActivities(username)

	if err != nil {
		return err
	}

	return activities.DisplayActivity(username, userActivities)
}
