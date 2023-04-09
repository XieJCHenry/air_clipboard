package clipboard

import (
	"context"
	"testing"

	"golang.design/x/clipboard"
)

func TestName(t *testing.T) {
	err := clipboard.Init()
	if err != nil {
		t.Fatalf("clipboard init failed, err=%s", err)
	}

	txtWatch := clipboard.Watch(context.TODO(), clipboard.FmtText)
	for data := range txtWatch {
		t.Logf("clipboard txt = %s", string(data))
	}

	//imgWatch := clipboard.Watch(context.TODO(), clipboard.FmtImage)
	//for data := range imgWatch {
	//	t.Logf("clipboard img size = %v", len(data))
	//}

}
