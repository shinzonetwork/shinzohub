package main

import (
	"log"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
)

// NewProbeCmd returns a cobra Command which tests the current node for liveness,
func NewProbeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        "probe",
		Short:                      "probe is a liveness probe which asserts that the node is consuming blocks and not stuck at block 0",
		SuggestionsMinimumDistance: 2,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			resp, err := clientCtx.Client.Status(cmd.Context())
			if err != nil {
				return err
			}
			height := resp.SyncInfo.LatestBlockHeight
			log.Printf("Latest block: %v", height)
			if height == 0 {
				log.Fatalf("Node liveness check failed: latest height %v", height)
			}

			return nil
		},
	}
	return cmd
}
