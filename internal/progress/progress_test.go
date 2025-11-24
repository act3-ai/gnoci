package progress

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"
	"time"

	"github.com/act3-ai/gnoci/internal/progress/progressmock"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestNewTicker(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			t.Helper()

			ctrl := gomock.NewController(t)
			evaluatorMock := progressmock.NewMockEvaluator(ctrl)

			expectedTotal := 10
			expectedDelta := 5
			evaluatorMock.EXPECT().
				Progress().
				Return(expectedTotal, expectedDelta, nil).
				MinTimes(1)

			ch := make(chan Progress)
			done := make(chan struct{})
			go func() {
				for p := range ch {
					assert.Equal(t, expectedTotal, p.Total)
					assert.Equal(t, expectedDelta, p.Delta)
				}
				close(done)
			}()

			ctx, cancel := context.WithCancel(t.Context())
			NewTicker(ctx, evaluatorMock, time.Nanosecond, ch)

			time.Sleep(time.Nanosecond * 5)
			synctest.Wait()
			cancel()
			<-done
		})
	})

	t.Run("Error", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			t.Helper()

			ctrl := gomock.NewController(t)
			evaluatorMock := progressmock.NewMockEvaluator(ctrl)

			expectedTotal := 10
			expectedDelta := 5
			expectedErr := errors.New("progress error")
			evaluatorMock.EXPECT().
				Progress().
				Return(expectedTotal, expectedDelta, expectedErr).
				MinTimes(1)

			ch := make(chan Progress)
			done := make(chan struct{})
			go func() {
				for p := range ch {
					// ensure progress messages are sent, despite errors
					assert.Equal(t, expectedTotal, p.Total)
					assert.Equal(t, expectedDelta, p.Delta)
				}
				close(done)
			}()

			ctx, cancel := context.WithCancel(t.Context())
			NewTicker(ctx, evaluatorMock, time.Nanosecond, ch)

			time.Sleep(time.Nanosecond * 5)
			synctest.Wait()
			cancel()
			<-done
		})
	})
}
