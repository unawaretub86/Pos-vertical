package main

import (
	"encoding/binary"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"unicode/utf16"
)

// taskName is the per-user scheduled task that starts the POS at logon (US-089).
const taskName = "POS-Vertical"

// schtasksRunner runs schtasks.exe with args and returns its combined output
// and exit code. It is a seam so tests never touch the real Task Scheduler.
type schtasksRunner func(args ...string) (output string, exitCode int)

func realSchtasks(args ...string) (string, int) {
	out, err := exec.Command("schtasks.exe", args...).CombinedOutput()
	if err == nil {
		return string(out), 0
	}
	if ee, ok := err.(*exec.ExitError); ok {
		return string(out), ee.ExitCode()
	}
	return err.Error(), 1
}

// runAutostartFlag handles --install-autostart / --uninstall-autostart and
// returns the process exit code.
func runAutostartFlag(install bool, out io.Writer) int {
	if !install {
		return uninstallAutostart(realSchtasks, runtime.GOOS, out)
	}
	exe, err := os.Executable()
	if err == nil {
		exe, err = filepath.Abs(exe)
	}
	if err != nil {
		fmt.Fprintf(out, "No se pudo determinar la ruta de pos.exe: %v\n", err)
		return 1
	}
	return installAutostart(realSchtasks, exe, runtime.GOOS, out)
}

// resolvedDBPath returns the absolute path of the database file, which is
// what the process logs at startup.
func resolvedDBPath(p string) string {
	abs, err := filepath.Abs(p)
	if err != nil {
		return p
	}
	return abs
}

// splitWindowsPath splits on backslashes explicitly: filepath.Dir does not
// split `\` on Linux, where CI runs the tests.
func splitWindowsPath(p string) (dir string) {
	i := strings.LastIndex(p, `\`)
	if i < 0 {
		return ""
	}
	return p[:i]
}

func xmlEscape(s string) string {
	var b strings.Builder
	_ = xml.EscapeText(&b, []byte(s))
	return b.String()
}

// buildTaskXML returns the task definition (UTF-8, from the root <Task>):
// logon trigger, working directory = folder of the exe (the database path is
// relative to it), no time limit, one instance, runs on battery, restarts on
// failure 3 times a minute apart, normal user rights.
func buildTaskXML(exePath string) string {
	return buildTaskXMLFor(exePath, "")
}

// buildTaskXMLFor is buildTaskXML restricted to one Windows user (DOMAIN\name);
// with an empty user the trigger fires for any logon.
func buildTaskXMLFor(exePath, user string) string {
	userTag := ""
	if user != "" {
		userTag = "<UserId>" + xmlEscape(user) + "</UserId>"
	}
	return `<Task version="1.4" xmlns="http://schemas.microsoft.com/windows/2004/02/mit/task">
  <RegistrationInfo>
    <Description>Starts the Pos-vertical POS when the user logs in.</Description>
  </RegistrationInfo>
  <Triggers>
    <LogonTrigger>
      <Enabled>true</Enabled>
      ` + userTag + `
    </LogonTrigger>
  </Triggers>
  <Principals>
    <Principal id="Author">
      ` + userTag + `
      <LogonType>InteractiveToken</LogonType>
      <RunLevel>LeastPrivilege</RunLevel>
    </Principal>
  </Principals>
  <Settings>
    <MultipleInstancesPolicy>IgnoreNew</MultipleInstancesPolicy>
    <DisallowStartIfOnBatteries>false</DisallowStartIfOnBatteries>
    <StopIfGoingOnBatteries>false</StopIfGoingOnBatteries>
    <StartWhenAvailable>true</StartWhenAvailable>
    <ExecutionTimeLimit>PT0S</ExecutionTimeLimit>
    <RestartOnFailure>
      <Interval>PT1M</Interval>
      <Count>3</Count>
    </RestartOnFailure>
    <Enabled>true</Enabled>
  </Settings>
  <Actions Context="Author">
    <Exec>
      <Command>` + xmlEscape(exePath) + `</Command>
      <WorkingDirectory>` + xmlEscape(splitWindowsPath(exePath)) + `</WorkingDirectory>
    </Exec>
  </Actions>
</Task>`
}

// utf16LEWithBOM encodes s the way `schtasks /XML` expects.
func utf16LEWithBOM(s string) []byte {
	units := utf16.Encode([]rune(s))
	b := make([]byte, 2, 2+2*len(units))
	b[0], b[1] = 0xFF, 0xFE
	for _, u := range units {
		b = binary.LittleEndian.AppendUint16(b, u)
	}
	return b
}

func currentWindowsUser() string {
	name := os.Getenv("USERNAME")
	if name == "" {
		return ""
	}
	if domain := os.Getenv("USERDOMAIN"); domain != "" {
		return domain + `\` + name
	}
	return name
}

// installAutostart creates (or replaces, via /F) the logon task and returns
// the process exit code.
func installAutostart(run schtasksRunner, exePath, goos string, out io.Writer) int {
	if goos != "windows" {
		fmt.Fprintln(out, "El inicio automático solo está disponible en Windows")
		return 1
	}
	f, err := os.CreateTemp("", "pos-autostart-*.xml")
	if err != nil {
		fmt.Fprintf(out, "No se pudo preparar el inicio automático: %v\n", err)
		return 1
	}
	path := f.Name()
	defer os.Remove(path)

	content := `<?xml version="1.0" encoding="UTF-16"?>` + "\r\n" + buildTaskXMLFor(exePath, currentWindowsUser())
	_, werr := f.Write(utf16LEWithBOM(content))
	if cerr := f.Close(); werr == nil {
		werr = cerr
	}
	if werr != nil {
		fmt.Fprintf(out, "No se pudo preparar el inicio automático: %v\n", werr)
		return 1
	}

	output, code := run("/Create", "/TN", taskName, "/XML", path, "/F")
	if code != 0 {
		fmt.Fprintf(out, "No se pudo instalar el inicio automático (schtasks): %s\n", strings.TrimSpace(output))
		return 1
	}
	fmt.Fprintln(out, "Inicio automático instalado")
	return 0
}

// uninstallAutostart deletes the logon task; a task that does not exist is not
// an error.
func uninstallAutostart(run schtasksRunner, goos string, out io.Writer) int {
	if goos != "windows" {
		fmt.Fprintln(out, "El inicio automático solo está disponible en Windows")
		return 1
	}
	output, code := run("/Delete", "/TN", taskName, "/F")
	if code != 0 {
		lower := strings.ToLower(output)
		// English and Spanish Windows word the "not found" error differently.
		if strings.Contains(lower, "cannot find") || strings.Contains(lower, "no puede encontrar") || strings.Contains(lower, "does not exist") {
			fmt.Fprintln(out, "El inicio automático no estaba instalado")
			return 0
		}
		fmt.Fprintf(out, "No se pudo quitar el inicio automático (schtasks): %s\n", strings.TrimSpace(output))
		return 1
	}
	fmt.Fprintln(out, "Inicio automático quitado")
	return 0
}
