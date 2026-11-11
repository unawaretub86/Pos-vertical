package main

// US-089 contract §4, T-15..T-18 (Windows autostart via a scheduled task).
//
// API assumed (the contract names behaviour, not the Go symbols):
//
//	buildTaskXML(exePath string) string
//	    task definition as UTF-8 text; the file written for schtasks is UTF-16 LE with BOM.
//	    It must split a Windows path on backslashes whatever the host OS (CI runs on Linux).
//	type schtasksRunner func(args ...string) (output string, exitCode int)
//	installAutostart(run schtasksRunner, exePath, goos string, out io.Writer) int   // exit code
//	uninstallAutostart(run schtasksRunner, goos string, out io.Writer) int          // exit code
//	resolvedDBPath(p string) string                                                 // filepath.Abs

import (
	"bytes"
	"encoding/xml"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const posExe = `C:\POS & Co\pos.exe`

// T-15 (AC-1, AC-4)
func TestAutostart_T15_BuildTaskXML(t *testing.T) {
	x := buildTaskXML(posExe)

	dec := xml.NewDecoder(strings.NewReader(x))
	dec.CharsetReader = func(_ string, in io.Reader) (io.Reader, error) { return in, nil }
	var doc struct {
		XMLName xml.Name
	}
	if err := dec.Decode(&doc); err != nil {
		t.Fatalf("task XML does not parse: %v\n%s", err, x)
	}
	if doc.XMLName.Local != "Task" {
		t.Errorf("root element = %q, want Task", doc.XMLName.Local)
	}

	for _, want := range []string{
		`<Command>C:\POS &amp; Co\pos.exe</Command>`,
		`<WorkingDirectory>C:\POS &amp; Co</WorkingDirectory>`,
		`<LogonTrigger>`,
		`<ExecutionTimeLimit>PT0S</ExecutionTimeLimit>`,
		`<MultipleInstancesPolicy>IgnoreNew</MultipleInstancesPolicy>`,
		`<RestartOnFailure>`,
		`<Count>3</Count>`,
		`<Interval>PT1M</Interval>`,
		`<DisallowStartIfOnBatteries>false</DisallowStartIfOnBatteries>`,
	} {
		if !strings.Contains(x, want) {
			t.Errorf("task XML missing %s", want)
		}
	}
	if strings.Contains(x, "<Delay>") {
		t.Error("task XML has a <Delay>, want none")
	}
	if strings.Contains(x, "HighestAvailable") {
		t.Error("task XML requests elevation, want normal user rights")
	}
}

// fakeSchtasks records every invocation and, for /Create, the XML file it was given.
type fakeSchtasks struct {
	calls    [][]string
	xmlPaths []string
	xmlBytes [][]byte
	output   string
	exit     int
}

func (f *fakeSchtasks) run(args ...string) (string, int) {
	f.calls = append(f.calls, append([]string(nil), args...))
	for i, a := range args {
		if a == "/XML" && i+1 < len(args) {
			f.xmlPaths = append(f.xmlPaths, args[i+1])
			b, _ := os.ReadFile(args[i+1])
			f.xmlBytes = append(f.xmlBytes, b)
		}
	}
	return f.output, f.exit
}

// T-16 (AC-1, AC-2)
func TestAutostart_T16_InstallAndUninstallCalls(t *testing.T) {
	f := &fakeSchtasks{}
	var out bytes.Buffer
	if code := installAutostart(f.run, posExe, "windows", &out); code != 0 {
		t.Fatalf("install exit = %d, want 0 (%s)", code, out.String())
	}
	if !strings.Contains(out.String(), "Inicio automático instalado") {
		t.Errorf("install output %q lacks 'Inicio automático instalado'", out.String())
	}
	if len(f.calls) != 1 {
		t.Fatalf("install made %d schtasks calls, want 1: %v", len(f.calls), f.calls)
	}
	c := f.calls[0]
	if len(c) != 6 || c[0] != "/Create" || c[1] != "/TN" || c[2] != "POS-Vertical" || c[3] != "/XML" || c[5] != "/F" {
		t.Errorf("install call = %v, want /Create /TN POS-Vertical /XML <file> /F", c)
	}
	// The file given to schtasks is UTF-16 LE with BOM.
	if b := f.xmlBytes[0]; len(b) < 4 || b[0] != 0xFF || b[1] != 0xFE {
		t.Errorf("XML file does not start with the UTF-16 LE BOM: % x", b[:min(4, len(b))])
	}

	// Installing twice repeats the same call (idempotent through /F).
	if code := installAutostart(f.run, posExe, "windows", &out); code != 0 {
		t.Fatalf("second install exit = %d", code)
	}
	if len(f.calls) != 2 {
		t.Fatalf("after two installs: %d calls, want 2", len(f.calls))
	}
	a, b := f.calls[0], f.calls[1]
	if a[0] != b[0] || a[1] != b[1] || a[2] != b[2] || a[3] != b[3] || a[5] != b[5] {
		t.Errorf("second install differs: %v vs %v", a, b)
	}

	// Uninstall.
	f2 := &fakeSchtasks{}
	out.Reset()
	if code := uninstallAutostart(f2.run, "windows", &out); code != 0 {
		t.Fatalf("uninstall exit = %d, want 0", code)
	}
	if !strings.Contains(out.String(), "Inicio automático quitado") {
		t.Errorf("uninstall output %q lacks 'Inicio automático quitado'", out.String())
	}
	want := []string{"/Delete", "/TN", "POS-Vertical", "/F"}
	if len(f2.calls) != 1 || strings.Join(f2.calls[0], " ") != strings.Join(want, " ") {
		t.Errorf("uninstall calls = %v, want one call %v", f2.calls, want)
	}
}

// T-17 (AC-2, AC-3)
func TestAutostart_T17_Failures(t *testing.T) {
	// Delete of a task that does not exist is not an error.
	f := &fakeSchtasks{output: "ERROR: The system cannot find the file specified.", exit: 1}
	var out bytes.Buffer
	if code := uninstallAutostart(f.run, "windows", &out); code != 0 {
		t.Errorf("uninstall of a missing task exit = %d, want 0", code)
	}
	if !strings.Contains(out.String(), "no estaba instalado") {
		t.Errorf("output %q lacks 'no estaba instalado'", out.String())
	}

	// A failing create exits 1, carries the schtasks text and removes the temp XML.
	f = &fakeSchtasks{output: "ERROR: Access is denied.", exit: 1}
	out.Reset()
	if code := installAutostart(f.run, posExe, "windows", &out); code != 1 {
		t.Errorf("failing install exit = %d, want 1", code)
	}
	if !strings.Contains(out.String(), "Access is denied") {
		t.Errorf("output %q does not include the schtasks error text", out.String())
	}
	if len(f.xmlPaths) != 1 {
		t.Fatalf("create was not called with an XML file: %v", f.calls)
	}
	if _, err := os.Stat(f.xmlPaths[0]); err == nil {
		t.Errorf("temp XML %s still exists after a failed install", f.xmlPaths[0])
	}

	// The temp XML is also removed after a successful install.
	f = &fakeSchtasks{}
	out.Reset()
	if code := installAutostart(f.run, posExe, "windows", &out); code != 0 {
		t.Fatalf("install exit = %d", code)
	}
	if _, err := os.Stat(f.xmlPaths[0]); err == nil {
		t.Errorf("temp XML %s still exists after a successful install", f.xmlPaths[0])
	}

	// Not Windows: exit 1 without calling schtasks.
	f = &fakeSchtasks{}
	out.Reset()
	if code := installAutostart(f.run, posExe, "linux", &out); code != 1 {
		t.Errorf("install on linux exit = %d, want 1", code)
	}
	out.Reset()
	if code := uninstallAutostart(f.run, "linux", &out); code != 1 {
		t.Errorf("uninstall on linux exit = %d, want 1", code)
	}
	if len(f.calls) != 0 {
		t.Errorf("schtasks was called on linux: %v", f.calls)
	}
}

// T-18 (AC-5): the logged database path is absolute.
func TestResolvedDBPath_T18_IsAbsolute(t *testing.T) {
	want, err := filepath.Abs("data.db")
	if err != nil {
		t.Fatal(err)
	}
	if got := resolvedDBPath("data.db"); got != want {
		t.Errorf("resolvedDBPath(data.db) = %q, want %q", got, want)
	}
	if got := resolvedDBPath("data.db"); !filepath.IsAbs(got) {
		t.Errorf("resolvedDBPath(data.db) = %q is not absolute", got)
	}
}
