//go:build windows

package service

import (
	"context"
	"fmt"
	"syscall"
	"unsafe"
)

var (
	winspool         = syscall.NewLazyDLL("winspool.drv")
	procOpenPrinter  = winspool.NewProc("OpenPrinterW")
	procClosePrinter = winspool.NewProc("ClosePrinter")
	procStartDoc     = winspool.NewProc("StartDocPrinterW")
	procEndDoc       = winspool.NewProc("EndDocPrinter")
	procStartPage    = winspool.NewProc("StartPagePrinter")
	procEndPage      = winspool.NewProc("EndPagePrinter")
	procWritePrinter = winspool.NewProc("WritePrinter")
)

// docInfo1 mirrors DOC_INFO_1W.
type docInfo1 struct {
	docName    *uint16
	outputFile *uint16
	datatype   *uint16
}

// Send writes data to the named Windows print queue as a RAW job.
func (w WindowsTransport) Send(ctx context.Context, data []byte) error {
	name, err := syscall.UTF16PtrFromString(w.Name)
	if err != nil {
		return fmt.Errorf("printer name: %w", err)
	}
	var h syscall.Handle
	if r, _, e := procOpenPrinter.Call(uintptr(unsafe.Pointer(name)), uintptr(unsafe.Pointer(&h)), 0); r == 0 {
		return fmt.Errorf("open printer %q: %w", w.Name, e)
	}
	defer procClosePrinter.Call(uintptr(h))

	docName, _ := syscall.UTF16PtrFromString("POS ticket")
	raw, _ := syscall.UTF16PtrFromString("RAW")
	di := docInfo1{docName: docName, datatype: raw}
	if r, _, e := procStartDoc.Call(uintptr(h), 1, uintptr(unsafe.Pointer(&di))); r == 0 {
		return fmt.Errorf("start print job on %q: %w", w.Name, e)
	}
	defer procEndDoc.Call(uintptr(h))

	if r, _, e := procStartPage.Call(uintptr(h)); r == 0 {
		return fmt.Errorf("start page on %q: %w", w.Name, e)
	}
	defer procEndPage.Call(uintptr(h))

	if len(data) == 0 {
		return nil
	}
	var written uint32
	if r, _, e := procWritePrinter.Call(uintptr(h), uintptr(unsafe.Pointer(&data[0])), uintptr(len(data)), uintptr(unsafe.Pointer(&written))); r == 0 {
		return fmt.Errorf("write to %q: %w", w.Name, e)
	}
	if int(written) != len(data) {
		return fmt.Errorf("write to %q: short write (%d of %d bytes)", w.Name, written, len(data))
	}
	return nil
}
