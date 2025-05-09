package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"tokeping/pkg/config"
	"tokeping/pkg/daemon"
	_ "tokeping/plugins/dns"
	_ "tokeping/plugins/file"
	_ "tokeping/plugins/influxdb"
	_ "tokeping/plugins/ping"
	_ "tokeping/plugins/ws"
	_ "tokeping/plugins/zmq"
	_ "tokeping/plugins/mtr"
)

var cfgFile string

var rootCmd = &cobra.Command{
	Use:   "tokeping",
	Short: "Realtime latency graphing daemon",
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the tokeping daemon",
	Run: func(cmd *cobra.Command, args []string) {
		// Daemonize stuff
		if daemonize, _ := cmd.Flags().GetBool("daemonize"); daemonize {
			// 1️⃣ Remove the daemonize flag for the child
			args := []string{os.Args[0]}
			for _, a := range os.Args[1:] {
				if a == "--daemonize" || a == "-d" {
					continue
				}
				args = append(args, a)
			}

			// 2️⃣ Open /dev/null for stdio
			devNull, err := os.OpenFile("/dev/null", os.O_RDWR, 0)
			if err != nil {
				fmt.Fprintf(os.Stderr, "daemonize: open /dev/null: %v\n", err)
				os.Exit(1)
			}
			defer devNull.Close()

			// 3️⃣ Spawn a new session
			attr := &os.ProcAttr{
				Dir:   ".", // or your desired working dir
				Env:   os.Environ(),
				Files: []*os.File{devNull, devNull, devNull},
				Sys:   &syscall.SysProcAttr{Setsid: true},
			}

			proc, err := os.StartProcess(os.Args[0], args, attr)
			if err != nil {
				fmt.Fprintf(os.Stderr, "daemonize: start process failed: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("tokeping daemon started, PID %d\n", proc.Pid)
			os.Exit(0)
		}

		// Load config
		conf, err := config.Load(cfgFile)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		// Write PID file if configured
		if conf.PIDFile != "" {
			pid := []byte(fmt.Sprintf("%d", os.Getpid()))
			if err := os.WriteFile(conf.PIDFile, pid, 0644); err != nil {
				fmt.Fprintf(os.Stderr, "failed to write pid file: %v", err)
			}
		}

		// Create and run daemon
		d, err := daemon.New(conf)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer cancel()

		go d.Run(ctx)
		<-ctx.Done()
		d.Stop()
	},
}

func init() {
	cobra.OnInitialize(func() {
		viper.SetConfigFile(cfgFile)
	})
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "config.yaml", "config file")
	rootCmd.AddCommand(startCmd)
	startCmd.Flags().BoolP("daemonize", "d", false, "Run in background as daemon")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
