/*
Copyright 2025 Vigil Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package vm

import (
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/creack/pty"
	"golang.org/x/crypto/ssh"
)

// LocalShellSession provides a local shell session with a PTY,
// offering an interface compatible with *ssh.Session.
type LocalShellSession struct {
	cmd *exec.Cmd
	pt  *os.File
}

// NewLocalShellSession starts a new local shell (/bin/bash -l) with a PTY.
func NewLocalShellSession() (*LocalShellSession, error) {
	cmd := exec.Command("/bin/bash", "-l")
	pt, err := pty.Start(cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to start local shell: %w", err)
	}
	return &LocalShellSession{cmd: cmd, pt: pt}, nil
}

// StdinPipe returns the PTY file as a writer.
func (s *LocalShellSession) StdinPipe() (io.WriteCloser, error) {
	return s.pt, nil
}

// StdoutPipe returns the PTY file as a reader.
func (s *LocalShellSession) StdoutPipe() (io.Reader, error) {
	return s.pt, nil
}

// StderrPipe returns the PTY file as a reader (PTY merges stderr into stdout).
func (s *LocalShellSession) StderrPipe() (io.Reader, error) {
	return s.pt, nil
}

// RequestPty sets the initial terminal size; the PTY is already created.
func (s *LocalShellSession) RequestPty(term string, h, w int, modes ssh.TerminalModes) error {
	return pty.Setsize(s.pt, &pty.Winsize{Rows: uint16(h), Cols: uint16(w)})
}

// WindowChange resizes the PTY to the given rows and columns.
func (s *LocalShellSession) WindowChange(rows, cols int) error {
	return pty.Setsize(s.pt, &pty.Winsize{Rows: uint16(rows), Cols: uint16(cols)})
}

// Shell is a no-op because the shell is already started in NewLocalShellSession.
func (s *LocalShellSession) Shell() error {
	return nil
}

// Close kills the shell process and closes the PTY.
func (s *LocalShellSession) Close() error {
	s.pt.Close()
	if s.cmd.Process != nil {
		s.cmd.Process.Kill()
	}
	return s.cmd.Wait()
}
