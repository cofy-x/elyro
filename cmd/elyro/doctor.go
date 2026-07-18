package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"

	"github.com/cofy-x/elyro/internal/cliui"
	elyroversion "github.com/cofy-x/elyro/internal/version"
	elyroworkspace "github.com/cofy-x/elyro/internal/workspace"
	"github.com/spf13/cobra"
)

func newDoctorCmd() *cobra.Command {
	var outputJSON bool
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Check local Elyro prerequisites and configuration",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if outputJSON {
				return runDoctorJSON(cmd.OutOrStdout())
			}
			return runDoctor(cmd.OutOrStdout())
		},
	}
	cmd.Flags().BoolVar(&outputJSON, "json", false, "Print checks as JSON")
	return cmd
}

func runDoctor(out io.Writer) error {
	return printDoctorChecks(out, "Elyro doctor:", doctorChecks())
}

type doctorJSONCheck struct {
	Name     string `json:"name"`
	Status   string `json:"status"`
	Required bool   `json:"required"`
	Message  string `json:"message,omitempty"`
}

type doctorJSONView struct {
	SchemaVersion int               `json:"schema_version"`
	Healthy       bool              `json:"healthy"`
	Checks        []doctorJSONCheck `json:"checks"`
}

func runDoctorJSON(out io.Writer) error {
	checks := doctorChecks()
	view := doctorJSONView{SchemaVersion: 1, Healthy: true, Checks: make([]doctorJSONCheck, 0, len(checks))}
	for _, check := range checks {
		item := doctorJSONCheck{Name: check.name, Required: check.required, Status: "ok"}
		if check.err != nil {
			item.Message = check.err.Error()
			if check.required {
				item.Status = "fail"
				view.Healthy = false
			} else {
				item.Status = "warn"
			}
		}
		view.Checks = append(view.Checks, item)
	}
	encoder := json.NewEncoder(out)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(view); err != nil {
		return err
	}
	if !view.Healthy {
		return fmt.Errorf("one or more checks failed")
	}
	return nil
}

func doctorChecks() []doctorCheck {
	return []doctorCheck{
		{name: "docker", required: true, err: checkCommand("docker")},
		{name: "ssh", required: true, err: checkCommand("ssh")},
		{name: "go", required: !elyroversion.IsRelease(), err: checkCommand("go")},
		{name: "docker daemon", required: true, err: checkDockerDaemon()},
		{name: "workspace registry", required: false, err: checkWorkspaceRegistry()},
	}
}

type doctorCheck struct {
	name     string
	required bool
	err      error
}

func printDoctorChecks(out io.Writer, title string, checks []doctorCheck) error {
	ok := true
	ui := cliui.New(out)
	if err := ui.Title(strings.TrimSuffix(title, ":")); err != nil {
		return err
	}
	for _, check := range checks {
		if check.err != nil {
			if check.required {
				ok = false
				if err := ui.Failure(fmt.Sprintf("%s: %v", check.name, check.err)); err != nil {
					return err
				}
			} else {
				if err := ui.Warning(fmt.Sprintf("%s: %v", check.name, check.err)); err != nil {
					return err
				}
			}
			continue
		}
		if err := ui.Success(check.name); err != nil {
			return err
		}
	}
	if !ok {
		return fmt.Errorf("one or more checks failed")
	}
	return nil
}

func checkCommand(name string) error {
	_, err := exec.LookPath(name)
	return err
}

func checkDockerDaemon() error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "docker", "info", "--format", "{{.ServerVersion}}")
	out, err := cmd.CombinedOutput()
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			return err
		}
		return fmt.Errorf("%w: %s", err, msg)
	}
	return nil
}

func checkWorkspaceRegistry() error {
	path, err := elyroworkspace.DefaultPath()
	if err != nil {
		return err
	}
	store, err := elyroworkspace.Load(path)
	if err != nil {
		return err
	}
	if len(store.Workspaces) == 0 {
		return fmt.Errorf("none found; run `elyro up` from a project")
	}
	return nil
}
