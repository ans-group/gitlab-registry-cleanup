package progress

import (
	"github.com/cheggaaa/pb"
)

type Progress struct {
	bar     *pb.ProgressBar
	enabled bool
}

func NewProgress(enabled bool, max int) *Progress {
	return &Progress{
		bar:     pb.New(max),
		enabled: enabled,
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
