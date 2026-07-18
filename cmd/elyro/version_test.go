package main

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestVersionCommandJSON(t *testing.T) {
	cmd := newVersionCmd()
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetArgs([]string{"--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	var view versionView
	if err := json.Unmarshal(output.Bytes(), &view); err != nil {
		t.Fatal(err)
	}
	if view.Version == "" || view.Commit == "" || view.BuildDate == "" {
		t.Fatalf("version view = %+v", view)
	}
}
