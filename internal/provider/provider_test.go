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

func TestFunctions(t *testing.T) {
	p := New().(*BurnhamProvider)
	funcs := p.Functions(context.Background())
	if len(funcs) != 57 {
		t.Errorf("expected 57 functions, got %d", len(funcs))
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
