package source

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/stretchr/testify/assert"
)

func Test_StaticImageSourceName(t *testing.T) {
	src := NewStaticImageSource([]string{}, 0)
	assert.Equal(t, "static", src.Name())
}

func Test_StaticImageSource(t *testing.T) {
	src := NewStaticImageSource([]string{"foo", "bar", "baz"}, 0)
	staticSource := src.(*StaticImageSource)
	staticSource.clock = clock.NewMock()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go staticSource.Run(ctx)

	var emitted []string

	for image := range src.ImageCh() {
		emitted = append(emitted, image)
	}

	time.Sleep(time.Millisecond * 10)
	assert.ElementsMatch(t, emitted, []string{"foo", "bar", "baz"})
}

func Test_StaticImageSourceWithResync(t *testing.T) {
	src := NewStaticImageSource([]string{"foo", "bar", "baz"}, time.Minute*2)
	staticSource := src.(*StaticImageSource)
	mockClock := clock.NewMock()
	staticSource.clock = mockClock

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go staticSource.Run(ctx)

	wg := sync.WaitGroup{}
	wg.Add(3)

	var emitted []string

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case image := <-src.ImageCh():
				emitted = append(emitted, image)
				wg.Done()
			}
		}
	}()

	wg.Wait()
	time.Sleep(time.Millisecond * 10)
	assert.ElementsMatch(t, emitted, []string{"foo", "bar", "baz"})

	wg.Add(3)
	mockClock.Add(time.Minute * 3)
	wg.Wait()

	assert.EqualValues(t, emitted, []string{"foo", "bar", "baz", "foo", "bar", "baz"})
}
