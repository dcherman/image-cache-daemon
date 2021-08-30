package source

import (
	"context"
)

type ImageSource interface {
	ImageCh() <-chan string
	Images() []string
	Name() string
	Run(context.Context)
}
