package i18n

import (
	"testing"
)

type ExampleResource struct {
	Name string `toml:"name"`
}

type ExampleResources = genericResources[string, ExampleResource]

func TestLoad(t *testing.T) {
	resources := ExampleResources{}
	if err := load("testdata", resources); err != nil {
		t.Fatalf("Failed to load example resources: %v", err)
	}

	if len(resources) == 0 {
		t.Error("No example resources loaded")
	}
}

func TestGet(t *testing.T) {
	resources := ExampleResources{}
	if err := load("testdata", resources); err != nil {
		t.Fatalf("Failed to load example resources: %v", err)
	}

	rsTest, ok := resources.Get("test")
	if !ok {
		t.Error("Expected resource 'test' to exist")
	}
	if rsTest.Name != "test-generic" {
		t.Errorf("rsTest.Name = %s, expected 'test-generic'", rsTest.Name)
	}

	rsTestAlpha, ok := resources.Get("test-ALPHA")
	if !ok {
		t.Error("Expected resource 'test-ALPHA' to exist")
	}
	if rsTestAlpha.Name != "test-ALPHA" {
		t.Errorf("rsTestOne.Name = %s, expected 'test-ALPHA'", rsTestAlpha.Name)
	}

	_, ok = resources.Get("test-non-existent")
	if ok {
		t.Error("Expected resource 'test-non-existent' to not exist")
	}
}

func TestGetOrGeneric(t *testing.T) {
	resources := ExampleResources{}
	if err := load("testdata", resources); err != nil {
		t.Fatalf("Failed to load example resources: %v", err)
	}

	rsTest, ok := resources.GetOrGeneric("test")
	if !ok {
		t.Error("Expected resource 'test' to exist")
	}
	if rsTest.Name != "test-generic" {
		t.Errorf("rsTest.Name = %s, expected 'test-generic'", rsTest.Name)
	}

	rsTestAlpha, ok := resources.GetOrGeneric("test-ALPHA")
	if !ok {
		t.Error("Expected resource 'test-ALPHA' to exist")
	}
	if rsTestAlpha.Name != "test-ALPHA" {
		t.Errorf("rsTestAlpha.Name = %s, expected 'test-ALPHA'", rsTestAlpha.Name)
	}

	rsNonExistent, ok := resources.GetOrGeneric("test-non-existent")
	if !ok {
		t.Error("Expected resource 'test-non-existent' to exist")
	}
	if rsNonExistent.Name != "test-generic" {
		t.Errorf("rsNonExistent.Name = %s, expected 'test-generic'", rsNonExistent.Name)
	}
}
