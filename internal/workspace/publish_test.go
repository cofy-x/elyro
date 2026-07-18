package workspace

import "testing"

func TestParsePublishSpecs(t *testing.T) {
	t.Parallel()

	publishes, err := ParsePublishSpecs([]string{"8080:8000", "9229"})
	if err != nil {
		t.Fatalf("ParsePublishSpecs returned error: %v", err)
	}
	if len(publishes) != 2 {
		t.Fatalf("expected 2 publishes, got %d", len(publishes))
	}
	if got, want := NormalizePublishSpecs(publishes), "8080:8000,9229"; got != want {
		t.Fatalf("NormalizePublishSpecs mismatch: got %q want %q", got, want)
	}
}

func TestParsePublishSpecsRejectsInvalidValues(t *testing.T) {
	t.Parallel()

	cases := []string{"abc", "1:2:3", "70000", "0"}
	for _, input := range cases {
		if _, err := ParsePublishSpecs([]string{input}); err == nil {
			t.Fatalf("expected parse error for %q", input)
		}
	}
}

func TestMergePortPublishesSortsAndDeduplicates(t *testing.T) {
	t.Parallel()

	environment, _ := ParsePublishSpecs([]string{"8000", "9000:9001"})
	command, _ := ParsePublishSpecs([]string{"8000", "7000:7001"})
	merged, err := MergePortPublishes(environment, command)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := NormalizePublishSpecs(merged), "7000:7001,8000,9000:9001"; got != want {
		t.Fatalf("merged publishes = %q, want %q", got, want)
	}
}

func TestMergePortPublishesRejectsHostPortConflict(t *testing.T) {
	t.Parallel()

	first, _ := ParsePublishSpecs([]string{"8000"})
	second, _ := ParsePublishSpecs([]string{"8000:9000"})
	if _, err := MergePortPublishes(first, second); err == nil {
		t.Fatal("MergePortPublishes() error = nil")
	}
}
