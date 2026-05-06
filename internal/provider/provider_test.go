package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/provider"
)

func TestNew(t *testing.T) {
	p := New()
	if p == nil {
		t.Fatal("New() returned nil")
	}
}

func TestMetadata(t *testing.T) {
	p := New()
	resp := &provider.MetadataResponse{}
	p.Metadata(context.Background(), provider.MetadataRequest{}, resp)
	if resp.TypeName != "burnham" {
		t.Errorf("expected TypeName='burnham', got %q", resp.TypeName)
	}
}

// TestFunctions checks the aggregator is wired up. We don't assert an exact
// count: the per-function acceptance tests (acceptance_*_test.go) already
// catch missing/broken registrations through the protocol layer, and pinning
// the count here would just be ceremony to update on every new function.
func TestFunctions(t *testing.T) {
	p := New().(*BurnhamProvider)
	funcs := p.Functions(context.Background())
	if len(funcs) == 0 {
		t.Error("Functions() returned empty list — aggregator likely not wired up")
	}
}

func TestResources(t *testing.T) {
	p := New().(*BurnhamProvider)
	if r := p.Resources(context.Background()); r != nil {
		t.Errorf("expected nil Resources, got %v", r)
	}
}

func TestDataSources(t *testing.T) {
	p := New().(*BurnhamProvider)
	if d := p.DataSources(context.Background()); d != nil {
		t.Errorf("expected nil DataSources, got %v", d)
	}
}
