package sub

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/cybozu-go/well"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"
	"github.com/ymmt2005/pdf-converter/converter"
)

var config struct {
	maxLength      int64
	maxConvertTime time.Duration
	maxParallel    int
	bindAddr       string
}

var rootCmd = &cobra.Command{
	Use:   "pdf-converter WORKDIR",
	Short: "run HTTP server to convert Office files to PDF",
	Long:  `run HTTP server to convert Office files to PDF`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true

		dir := args[0]
		fi, err := os.Stat(dir)
		if err != nil {
			return err
		}
		if !fi.IsDir() {
			return fmt.Errorf("%s is not a directory", dir)
		}

		mux := http.NewServeMux()
		mux.Handle("/metrics", promhttp.Handler())
		mux.Handle("/convert", NewConvertHandler(
			converter.NewConverter(),
			dir,
			config.maxLength,
			config.maxConvertTime,
			config.maxParallel,
		))
		s := well.HTTPServer{
			Server: &http.Server{
				Addr:    config.bindAddr,
				Handler: mux,
			},
		}
		if err := s.ListenAndServe(); err != nil {
			return err
		}
		well.Stop()
		return well.Wait()
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	pf := rootCmd.PersistentFlags()
	pf.Int64Var(&config.maxLength, "max-length", 1<<30, "the maximum length of the uploaded contents")
	pf.DurationVar(&config.maxConvertTime, "max-convert-time", 3*time.Minute, "the maximum time allowed for conversion")
	pf.IntVar(&config.maxParallel, "max-parallel", 0, "the maximum parallel conversions.  0 for unlimited parallelism")
	pf.StringVar(&config.bindAddr, "listen", ":8080", "bind address of the HTTP server")
}
