package progress

import (
	"github.com/cheggaaa/pb"
	"github.com/spf13/cobra"
)

type Progress struct {
	bar     *pb.ProgressBar
	enabled bool
}

func NewProgress(cmd *cobra.Command, max int) *Progress {
	progress, _ := cmd.Flags().GetBool("progress")
	return &Progress{
		bar:     pb.New(max),
		enabled: progress,
	}
}

func (p *Progress) Start() {
	if p.enabled {
		p.bar.Start()
	}
}

func (p *Progress) Increment() {
	if p.enabled {
		p.bar.Increment()
	}

}

func (p *Progress) Finish() {
	if p.enabled {
		p.bar.Finish()
	}
}
