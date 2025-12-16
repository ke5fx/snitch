package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"snitch/internal/collector"
	"syscall"
	"time"

	"github.com/spf13/cobra"
)

var (
	watchInterval time.Duration
	watchCount    int
)

var watchCmd = &cobra.Command{
	Use:   "watch [filters...]",
	Short: "Stream connection events as json frames",
	Long: `Stream connection events as json frames.
	
Filters are specified in key=value format. For example:
  snitch watch proto=tcp state=established

Available filters:
  proto, state, pid, proc, lport, rport, user, laddr, raddr, contains
`,
	Run: func(cmd *cobra.Command, args []string) {
		runWatchCommand(args)
	},
}

func runWatchCommand(args []string) {
	filters, err := parseFilters(args)
	if err != nil {
		log.Fatalf("Error parsing filters: %v", err)
	}
	filters.IPv4 = ipv4
	filters.IPv6 = ipv6

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle interrupts gracefully
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		cancel()
	}()

	ticker := time.NewTicker(watchInterval)
	defer ticker.Stop()

	count := 0
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			connections, err := collector.GetConnections()
			if err != nil {
				log.Printf("Error getting connections: %v", err)
				continue
			}

			filteredConnections := collector.FilterConnections(connections, filters)
			
			frame := map[string]interface{}{
				"timestamp":   time.Now().Format(time.RFC3339Nano),
				"connections": filteredConnections,
				"count":       len(filteredConnections),
			}

			jsonOutput, err := json.Marshal(frame)
			if err != nil {
				log.Printf("Error marshaling JSON: %v", err)
				continue
			}

			fmt.Println(string(jsonOutput))

			count++
			if watchCount > 0 && count >= watchCount {
				return
			}
		}
	}
}

func init() {
	rootCmd.AddCommand(watchCmd)
	watchCmd.Flags().DurationVarP(&watchInterval, "interval", "i", time.Second, "Refresh interval (e.g., 500ms, 2s)")
	watchCmd.Flags().IntVarP(&watchCount, "count", "c", 0, "Number of frames to emit (0 = unlimited)")
	watchCmd.Flags().BoolVarP(&ipv4, "ipv4", "4", false, "Only show IPv4 connections")
	watchCmd.Flags().BoolVarP(&ipv6, "ipv6", "6", false, "Only show IPv6 connections")
}
