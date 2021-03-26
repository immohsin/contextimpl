package contextimpl

import (
	"testing"
	"time"
)

func TestBackgroundNotTODO(t *testing.T) {
	todo := TODO()
	bg := Background()

	if todo == bg {
		t.Errorf("TODO and Background are equal: %p vs %p", todo, bg)
	}
}

func TestCanceledContext(t *testing.T) {
	ctx, cancel := WithCancel(Background())
	time.AfterFunc(1*time.Second, cancel)

	if err := ctx.Err(); err != nil {
		t.Errorf("error should be nil first, got %v", err)
	}
	<-ctx.Done()
	if err := ctx.Err(); err != Canceled {
		t.Errorf("error should be canceled now, got %v", err)
	}
}

func TestCanceledWhenParentCanceled(t *testing.T) {
	ctxA, cancelA := WithCancel(Background())
	ctxB, _ := WithCancel(ctxA)
	ctxC, _ := WithCancel(ctxB)

	cancelA()

	select {
	case <-ctxC.Done():
	case <-time.After(1 * time.Second):
		t.Errorf("time out")
	}

	if err := ctxC.Err(); err != Canceled {
		t.Errorf("error should be canceled now, got %v", err)
	}
}
