package source

import (
	"context"
	"time"

	"github.com/benbjohnson/clock"
)

type StaticImageSource struct {
	resyncPeriod time.Duration
	images       []string
	imageCh      chan string
	clock        clock.Clock
}

func (StaticImageSource) Name() string {
	return "static"
}

func (sis *StaticImageSource) ImageCh() <-chan string {
	return sis.imageCh
}

func (sis *StaticImageSource) Run(ctx context.Context) {
	for {
		for _, i := range sis.images {
			sis.imageCh <- i
		}

		if sis.resyncPeriod == 0 {
			break
		}

		sis.clock.Sleep(sis.resyncPeriod)
	}

	close(sis.imageCh)
}

func NewStaticImageSource(images []string, resyncPeriod time.Duration) ImageSource {
	return &StaticImageSource{
		imageCh:      make(chan string),
		images:       images,
		clock:        clock.New(),
		resyncPeriod: resyncPeriod,
	}
}
