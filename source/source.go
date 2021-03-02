package source

import (
	"context"
)

type ImageSource interface {
	ImageCh() <-chan string
	Name() string
	Run(context.Context)
}
