package editor

import (
	"errors"
	"testing"
)

func TestRemoteSSHOpenURI(t *testing.T) {
	t.Parallel()

	got := RemoteSSHOpenURI("vscode", "elyro-demo-abc", "/home/elyro/demo")
	want := "vscode://vscode-remote/ssh-remote+elyro-demo-abc/home/elyro/demo"
	if got != want {
		t.Fatalf("RemoteSSHOpenURI = %q, want %q", got, want)
	}

	gotEnc := RemoteSSHOpenURI("cursor", "dev box", "/home/elyro/dev box")
	wantEnc := "cursor://vscode-remote/ssh-remote+dev%20box/home/elyro/dev%20box"
	if gotEnc != wantEnc {
		t.Fatalf("RemoteSSHOpenURI encoded = %q, want %q", gotEnc, wantEnc)
	}
}

func TestRemoteSSHFolderURI(t *testing.T) {
	t.Parallel()

	got := RemoteSSHFolderURI("elyro-demo-abc", "/home/elyro/demo")
	want := "vscode-remote://ssh-remote+elyro-demo-abc/home/elyro/demo"
	if got != want {
		t.Fatalf("RemoteSSHFolderURI = %q, want %q", got, want)
	}

	gotEnc := RemoteSSHFolderURI("dev box", "/home/elyro/dev box")
	wantEnc := "vscode-remote://ssh-remote+dev%20box/home/elyro/dev%20box"
	if gotEnc != wantEnc {
		t.Fatalf("RemoteSSHFolderURI encoded = %q, want %q", gotEnc, wantEnc)
	}
}

func TestNewWindowCommand(t *testing.T) {
	t.Parallel()

	got := NewWindowCommand("code", "elyro-demo-abc", "/home/elyro/demo")
	want := "code --new-window --folder-uri 'vscode-remote://ssh-remote+elyro-demo-abc/home/elyro/demo'"
	if got != want {
		t.Fatalf("NewWindowCommand = %q, want %q", got, want)
	}
}

func TestDetectOptionsWithLookPath(t *testing.T) {
	t.Parallel()

	lookPath := func(bin string) (string, error) {
		switch bin {
		case "cursor":
			return "/usr/local/bin/cursor", nil
		case "code":
			return "", errors.New("not found")
		default:
			return "", errors.New("not found")
		}
	}

	options := DetectOptionsWithLookPath(lookPath, "elyro-demo", "/home/elyro/demo")
	if got, want := len(options), 1; got != want {
		t.Fatalf("expected %d option, got %d", want, got)
	}
	if got, want := options[0].Label, "Cursor"; got != want {
		t.Fatalf("label = %q, want %q", got, want)
	}
	if got, want := options[0].Command, "cursor --new-window --folder-uri 'vscode-remote://ssh-remote+elyro-demo/home/elyro/demo'"; got != want {
		t.Fatalf("command = %q, want %q", got, want)
	}
}
