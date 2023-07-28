package cmd

import (
	sendmail "github.com/axllent/mailpit/sendmail/cmd"
	"github.com/spf13/cobra"
)

var (
	//smtpAddr = "localhost:1025"
	//fromAddr string
	emailSize = 128
	concurrencyNum = 8
	perConcurrencyCnt = 128
	reuseConnection = false
)

// perfCmd
var perfCmd = &cobra.Command{
	Use:   "perf",
	Short: "perf",
	Long: `A perf command replacement.
	
You can optionally create a symlink called 'sendmail' to the main binary.`,
	Run: func(_ *cobra.Command, _ []string) {
		sendmail.Perf()
	},
}

func init() {
	rootCmd.AddCommand(perfCmd)

	// these are simply repeated for cli consistency
	perfCmd.Flags().StringVar(&smtpAddr, "smtp-addr", smtpAddr, "SMTP server address")
	perfCmd.Flags().StringVarP(&fromAddr, "from", "f", "", "SMTP sender")
	perfCmd.Flags().IntVar(&emailSize, "emailsize", 128, "The mailsize")
	perfCmd.Flags().IntVar(&concurrencyNum, "concurrencyNum", 8, "The concurrencyNum")
	perfCmd.Flags().IntVar(&perConcurrencyCnt, "perConcurrencyCnt", 128, "The perConcurrencyCnt")
	perfCmd.Flags().BoolVar(&reuseConnection, "reuseConnection", false, "If to reuse the mail dial connection")
	perfCmd.Flags().BoolP("long-i", "i", false, "Ignored. This flag exists for sendmail compatibility.")
	perfCmd.Flags().BoolP("long-t", "t", false, "Ignored. This flag exists for sendmail compatibility.")
}
