package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestPrintDoctorChecksAllowsOptionalFailures(t *testing.T) {
	var output bytes.Buffer
	err := printDoctorChecks(&output, "Elyro doctor:", []doctorCheck{
		{name: "docker", required: true},
		{name: "workspace registry", required: false, err: errors.New("none configured")},
	})
	if err != nil {
		t.Fatalf("printDoctorChecks() error = %v, want optional warning", err)
	}
	if got := output.String(); !strings.Contains(got, "! workspace registry: none configured") {
		t.Fatalf("doctor output = %q, want optional warning", got)
	}
}

func TestDoctorJSONSchema(t *testing.T) {
	var output bytes.Buffer
	view := doctorJSONView{SchemaVersion: 1, Healthy: true, Checks: []doctorJSONCheck{{Name: "docker", Status: "ok", Required: true}}}
	if err := json.NewEncoder(&output).Encode(view); err != nil {
		t.Fatal(err)
	}
	var got doctorJSONView
	if err := json.Unmarshal(output.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.SchemaVersion != 1 || !got.Healthy || len(got.Checks) != 1 {
		t.Fatalf("doctor JSON = %#v", got)
	}
}

func TestPrintDoctorChecksRejectsRequiredFailure(t *testing.T) {
	var output bytes.Buffer
	err := printDoctorChecks(&output, "Elyro doctor:", []doctorCheck{
		{name: "docker", required: true, err: errors.New("not found")},
	})
	if err == nil {
		t.Fatal("printDoctorChecks() succeeded, want required failure")
	}
}
