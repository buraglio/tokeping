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
    _ "tokeping/plugins/ping"
    _ "tokeping/plugins/ws"
    _ "tokeping/plugins/influxdb"
    _ "tokeping/plugins/zmq"
    _ "tokeping/plugins/file"
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
                fmt.Fprintf(os.Stderr, "failed to write pid file: %v
", err)
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
}

func main() {
    if err := rootCmd.Execute(); err != nil {
        os.Exit(1)
    }
}
