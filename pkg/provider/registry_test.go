package provider

import (
	"context"
	"sort"
	"testing"

	"github.com/papercomputeco/sweeper/pkg/worker"
)

func TestRegisterAndGet(t *testing.T) {
	// Reset registry for test isolation.
	mu.Lock()
	saved := registry
	registry = map[string]Provider{}
	mu.Unlock()
	defer func() {
		mu.Lock()
		registry = saved
		mu.Unlock()
	}()

	p := Provider{
		Name: "test-provider",
		Kind: KindCLI,
		NewExec: func(cfg Config) worker.Executor {
			return func(ctx context.Context, task worker.Task) worker.Result {
				return worker.Result{Success: true}
			}
		},
	}
	Register(p)

	got, err := Get("test-provider")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Name != "test-provider" {
		t.Errorf("expected name test-provider, got %s", got.Name)
	}
	if got.Kind != KindCLI {
		t.Errorf("expected KindCLI, got %d", got.Kind)
	}
}

func TestGetUnknownProvider(t *testing.T) {
	_, err := Get("nonexistent-provider-xyz")
	if err == nil {
		t.Error("expected error for unknown provider")
	}
}

func TestAvailable(t *testing.T) {
	// Reset registry for test isolation.
	mu.Lock()
	saved := registry
	registry = map[string]Provider{}
	mu.Unlock()
	defer func() {
		mu.Lock()
		registry = saved
		mu.Unlock()
	}()

	Register(Provider{Name: "beta", Kind: KindAPI})
	Register(Provider{Name: "alpha", Kind: KindCLI})

	names := Available()
	expected := []string{"alpha", "beta"}
	if len(names) != len(expected) {
		t.Fatalf("expected %d providers, got %d", len(expected), len(names))
	}
	sort.Strings(names)
	for i, name := range names {
		if name != expected[i] {
			t.Errorf("expected %s at index %d, got %s", expected[i], i, name)
		}
	}
}
