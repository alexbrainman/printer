// Copyright 2013 The Go Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Windows printing.
package printer

import (
	"syscall"
	"unsafe"
)

const (
	PRINTER_DRIVER_XPS uint32 = 0x00000002
	RAW                       = "RAW"
	XPS_PASS                  = "XPS_PASS"
)

//go:generate go run mksyscall_windows.go -output zapi.go printer.go

type DOC_INFO_1 struct {
	DocName    *uint16
	OutputFile *uint16
	Datatype   *uint16
}

type PRINTER_INFO_5 struct {
	PrinterName              *uint16
	PortName                 *uint16
	Attributes               uint32
	DeviceNotSelectedTimeout uint32
	TransmissionRetryTimeout uint32
}

type DRIVER_INFO_8 struct {
	Version                  uint32
	Name                     *uint16
	Environment              *uint16
	DriverPath               *uint16
	DataFile                 *uint16
	ConfigFile               *uint16
	HelpFile                 *uint16
	DependentFiles           *uint16
	MonitorName              *uint16
	DefaultDataType          *uint16
	PreviousNames            *uint16
	DriverDate               syscall.Filetime
	DriverVersion            uint64
	MfgName                  *uint16
	OEMUrl                   *uint16
	HardwareID               *uint16
	Provider                 *uint16
	PrintProcessor           *uint16
	VendorSetup              *uint16
	ColorProfiles            *uint16
	InfPath                  *uint16
	PrinterDriverAttributes  uint32
	CoreDriverDependencies   *uint16
	MinInboxDriverVerDate    syscall.Filetime
	MinInboxDriverVerVersion uint32
}

const (
	PRINTER_ENUM_LOCAL       = 2
	PRINTER_ENUM_CONNECTIONS = 4
)

//sys	GetDefaultPrinter(buf *uint16, bufN *uint32) (err error) = winspool.GetDefaultPrinterW
//sys	ClosePrinter(h syscall.Handle) (err error) = winspool.ClosePrinter
//sys	OpenPrinter(name *uint16, h *syscall.Handle, defaults uintptr) (err error) = winspool.OpenPrinterW
//sys	StartDocPrinter(h syscall.Handle, level uint32, docinfo *DOC_INFO_1) (err error) = winspool.StartDocPrinterW
//sys	EndDocPrinter(h syscall.Handle) (err error) = winspool.EndDocPrinter
//sys	WritePrinter(h syscall.Handle, buf *byte, bufN uint32, written *uint32) (err error) = winspool.WritePrinter
//sys	StartPagePrinter(h syscall.Handle) (err error) = winspool.StartPagePrinter
//sys	EndPagePrinter(h syscall.Handle) (err error) = winspool.EndPagePrinter
//sys	EnumPrinters(flags uint32, name *uint16, level uint32, buf *byte, bufN uint32, needed *uint32, returned *uint32) (err error) = winspool.EnumPrintersW

func Default() (string, error) {
	b := make([]uint16, 3)
	n := uint32(len(b))
	err := GetDefaultPrinter(&b[0], &n)
	if err != nil {
		if err != syscall.ERROR_INSUFFICIENT_BUFFER {
			return "", err
		}
		b = make([]uint16, n)
		err = GetDefaultPrinter(&b[0], &n)
		if err != nil {
			return "", err
		}
	}
	return syscall.UTF16ToString(b), nil
}

func GetDefaultPrinterType() (string, error) {
	printerName, _ := Default()
	return GetPrinterType(printerName)
}

func GetPrinterType(printerName string) (string, error) {
	b := make([]byte, 1024*10)
	n := uint32(len(b))
	err := GetPrinterDriver(printerName, &b[0], n)
	if err != nil {
		if err != syscall.ERROR_INSUFFICIENT_BUFFER {
			return "", err
		}
		b = make([]byte, n)
		err = GetPrinterDriver(printerName, &b[0], n)
		if err != nil {
			return "", err
		}
	}
	di := (*DRIVER_INFO_8)(unsafe.Pointer(&b[0]))

	driverType := RAW

	if di.PrinterDriverAttributes&PRINTER_DRIVER_XPS == 2 {
		driverType = XPS_PASS
	}

	return driverType, nil
}

// ReadNames return printer names on the system
func ReadNames() ([]string, error) {
	const flags = PRINTER_ENUM_LOCAL | PRINTER_ENUM_CONNECTIONS
	var needed, returned uint32
	buf := make([]byte, 1)
	err := EnumPrinters(flags, nil, 5, &buf[0], uint32(len(buf)), &needed, &returned)
	if err != nil {
		if err != syscall.ERROR_INSUFFICIENT_BUFFER {
			return nil, err
		}
		buf = make([]byte, needed)
		err = EnumPrinters(flags, nil, 5, &buf[0], uint32(len(buf)), &needed, &returned)
		if err != nil {
			return nil, err
		}
	}
	ps := (*[1024]PRINTER_INFO_5)(unsafe.Pointer(&buf[0]))[:returned]
	names := make([]string, 0, returned)
	for _, p := range ps {
		v := (*[1024]uint16)(unsafe.Pointer(p.PrinterName))[:]
		names = append(names, syscall.UTF16ToString(v))
	}
	return names, nil
}

type Printer struct {
	h syscall.Handle
}

func Open(name string) (*Printer, error) {
	var p Printer
	// TODO: implement pDefault parameter
	err := OpenPrinter(&(syscall.StringToUTF16(name))[0], &p.h, 0)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (p *Printer) StartDocument(name, datatype string) error {
	d := DOC_INFO_1{
		DocName:    &(syscall.StringToUTF16(name))[0],
		OutputFile: nil,
		Datatype:   &(syscall.StringToUTF16(datatype))[0],
	}
	return StartDocPrinter(p.h, 1, &d)
}

func (p *Printer) Write(b []byte) (int, error) {
	var written uint32
	err := WritePrinter(p.h, &b[0], uint32(len(b)), &written)
	if err != nil {
		return 0, err
	}
	return int(written), nil
}

func (p *Printer) EndDocument() error {
	return EndDocPrinter(p.h)
}

func (p *Printer) StartPage() error {
	return StartPagePrinter(p.h)
}

func (p *Printer) EndPage() error {
	return EndPagePrinter(p.h)
}

func (p *Printer) Close() error {
	return ClosePrinter(p.h)
}
