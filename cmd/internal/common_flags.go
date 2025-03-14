package internal

import "github.com/spf13/cobra"

const HostFlagName = "host"
const LogsFlagName = "logs"

type CommonFlags struct {
	Host    string
	LogsDir string
}

func AddCommonFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().String("host", "", "The hostname of the service we're proxying to (ex: <server>.servicebus.windows.net)")
	cmd.PersistentFlags().String("logs", ".", "The directory to write any logs or trace files")

	_ = cmd.MarkPersistentFlagRequired(HostFlagName)
}

func ExtractCommonFlags(cmd *cobra.Command) (CommonFlags, error) {
	host, err := cmd.Flags().GetString(HostFlagName)

	if err != nil {
		return CommonFlags{}, err
	}

	logs, err := cmd.Flags().GetString(LogsFlagName)

	if err != nil {
		return CommonFlags{}, err
	}

	return CommonFlags{
		Host:    host,
		LogsDir: logs,
	}, nil
}
