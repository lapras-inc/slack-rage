package cmd

import (
	"github.com/h3poteto/slack-rage/bolt"
	"github.com/spf13/cobra"
)

type runBolt struct {
	threshold int
	period    int
	speakers  int
	channel   string
	verbose   bool
}

func runBoltCmd() *cobra.Command {
	r := &runBolt{}
	cmd := &cobra.Command{
		Use:   "bolt",
		Short: "Run bot using Bolt",
		Run:   r.run,
	}

	flags := cmd.Flags()
	flags.IntVarP(&r.threshold, "threshold", "t", 10, "Threshold for rage judgement.")
	flags.IntVarP(&r.period, "period", "p", 1200, "Observation period seconds for rage judgement. This CLI notify when there are more than threshold posts per period.")
	flags.IntVarP(&r.speakers, "speakers", "s", 3, "This CLI notify when more speakers are participating.")
	flags.StringVarP(&r.channel, "channel", "c", "random", "Notify channel.")
	flags.BoolVarP(&r.verbose, "verbose", "v", false, "Enable verbose mode")

	return cmd
}

func (r *runBolt) run(cmd *cobra.Command, args []string) {
	b := bolt.New(r.threshold, r.period, r.speakers, r.channel, r.verbose)
	b.Start()
}
