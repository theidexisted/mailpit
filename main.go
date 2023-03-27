package main

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/axllent/mailpit/cmd"
	sendmail "github.com/axllent/mailpit/sendmail/cmd"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/axllent/mailpit/utils/logger"
)

func main() {

	exec, err := os.Executable()
	if err != nil {
		panic(err)
	}

	go func() {
		logger.Log().Infof("[http] starting metrics server on http://localhost:2112/metrics")
		http.Handle("/metrics", promhttp.Handler())
		http.ListenAndServe(":2112", nil)
	}()

	// running directly
	if normalize(filepath.Base(exec)) == normalize(filepath.Base(os.Args[0])) {
		cmd.Execute()
	} else {
		// symlinked
		sendmail.Run()
	}
}

// Normalize returns a lowercase string stripped of the file extension (if exists).
// Used for detecting Windows commands which ignores letter casing and `.exe`.
// eg: "MaIlpIT.Exe" returns "mailpit"
func normalize(s string) string {
	s = strings.ToLower(s)

	return strings.TrimSuffix(s, filepath.Ext(s))
}
