package watcher

import (
	"testing"
)

func TestPackageCompilation(t *testing.T) {
	var _ IngestionRunner
	var w *Watcher
	_ = w
	t.Log("watcher package compiles successfully")
}
