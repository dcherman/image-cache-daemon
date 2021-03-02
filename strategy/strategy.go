package strategy

import (
	"context"
)

type PullStrategy interface {
	PullImage(context.Context, string) error

	ImagePullSuccessCh() <-chan string
	ImagePullErrorCh() <-chan string
}
