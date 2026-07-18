package editor

import (
	"fmt"
	"net/url"
	"os/exec"
	"strings"
)

type Option struct {
	Label      string
	BinaryName string
	FolderURI  string
	Command    string
}

// RemoteSSHOpenURI returns a vscode:// or cursor:// link that opens the remote folder over SSH.
func RemoteSSHOpenURI(editorScheme, hostAlias, remoteFolder string) string {
	host := url.PathEscape(hostAlias)
	folder := encodeRemoteFolderPath(remoteFolder)
	return fmt.Sprintf("%s://vscode-remote/ssh-remote+%s/%s", editorScheme, host, folder)
}

// RemoteSSHFolderURI returns a vscode-remote:// URI suitable for code/cursor --folder-uri.
func RemoteSSHFolderURI(hostAlias, remoteFolder string) string {
	host := url.PathEscape(hostAlias)
	folder := encodeRemoteFolderPath(remoteFolder)
	return fmt.Sprintf("vscode-remote://ssh-remote+%s/%s", host, folder)
}

func NewWindowCommand(editorBin, hostAlias, remoteFolder string) string {
	return fmt.Sprintf("%s --new-window --folder-uri %s", editorBin, shellQuote(RemoteSSHFolderURI(hostAlias, remoteFolder)))
}

func DetectOptions(hostAlias, mountDir string) []Option {
	return DetectOptionsWithLookPath(exec.LookPath, hostAlias, mountDir)
}

func DetectOptionsWithLookPath(lookPath func(string) (string, error), hostAlias, mountDir string) []Option {
	var options []Option
	for _, candidate := range []struct {
		label string
		bin   string
	}{
		{label: "Cursor", bin: "cursor"},
		{label: "VS Code", bin: "code"},
	} {
		if _, err := lookPath(candidate.bin); err != nil {
			continue
		}
		options = append(options, Option{
			Label:      candidate.label,
			BinaryName: candidate.bin,
			FolderURI:  RemoteSSHFolderURI(hostAlias, mountDir),
			Command:    NewWindowCommand(candidate.bin, hostAlias, mountDir),
		})
	}
	return options
}

func Open(option Option) error {
	cmd := exec.Command(option.BinaryName, "--new-window", "--folder-uri", option.FolderURI)
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}

func encodeRemoteFolderPath(remoteFolder string) string {
	parts := strings.Split(strings.TrimPrefix(remoteFolder, "/"), "/")
	for i, part := range parts {
		parts[i] = url.PathEscape(part)
	}
	return strings.Join(parts, "/")
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'"'"'`) + "'"
}
