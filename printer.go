// Copyright 2013 The Go Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Windows printing.
package printer

import (
	"strings"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
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

type JOB_INFO_1 struct {
	JobID        uint32
	PrinterName  *uint16
	MachineName  *uint16
	UserName     *uint16
	Document     *uint16
	DataType     *uint16
	Status       *uint16
	StatusCode   uint32
	Priority     uint32
	Position     uint32
	TotalPages   uint32
	PagesPrinted uint32
	Submitted    syscall.Systemtime
}

const (
	PRINTER_ENUM_LOCAL       = 2
	PRINTER_ENUM_CONNECTIONS = 4

	PRINTER_DRIVER_XPS = 0x00000002
)

const (
	JOB_STATUS_PAUSED            = 0x00000001 // Job is paused
	JOB_STATUS_ERROR             = 0x00000002 // An error is associated with the job
	JOB_STATUS_DELETING          = 0x00000004 // Job is being deleted
	JOB_STATUS_SPOOLING          = 0x00000008 // Job is spooling
	JOB_STATUS_PRINTING          = 0x00000010 // Job is printing
	JOB_STATUS_OFFLINE           = 0x00000020 // Printer is offline
	JOB_STATUS_PAPEROUT          = 0x00000040 // Printer is out of paper
	JOB_STATUS_PRINTED           = 0x00000080 // Job has printed
	JOB_STATUS_DELETED           = 0x00000100 // Job has been deleted
	JOB_STATUS_BLOCKED_DEVQ      = 0x00000200 // Printer driver cannot print the job
	JOB_STATUS_USER_INTERVENTION = 0x00000400 // User action required
	JOB_STATUS_RESTART           = 0x00000800 // Job has been restarted
	JOB_STATUS_COMPLETE          = 0x00001000 // Job has been delivered to the printer
	JOB_STATUS_RETAINED          = 0x00002000 // Job has been retained in the print queue
	JOB_STATUS_RENDERING_LOCALLY = 0x00004000 // Job rendering locally on the client
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
//sys	GetPrinterDriver(h syscall.Handle, env *uint16, level uint32, di *byte, n uint32, needed *uint32) (err error) = winspool.GetPrinterDriverW
//sys	EnumJobs(h syscall.Handle, firstJob uint32, noJobs uint32, level uint32, buf *byte, bufN uint32, bytesNeeded *uint32, jobsReturned *uint32) (err error) = winspool.EnumJobsW

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
	ps := (*[1024]PRINTER_INFO_5)(unsafe.Pointer(&buf[0]))[:returned:returned]
	names := make([]string, 0, returned)
	for _, p := range ps {
		names = append(names, windows.UTF16PtrToString(p.PrinterName))
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

// DriverInfo stores information about printer driver.
type DriverInfo struct {
	Name        string
	Environment string
	DriverPath  string
	Attributes  uint32
}

// JobInfo stores information about a print job.
type JobInfo struct {
	JobID           uint32
	UserMachineName string
	UserName        string
	DocumentName    string
	DataType        string
	Status          string
	StatusCode      uint32
	Priority        uint32
	Position        uint32
	TotalPages      uint32
	PagesPrinted    uint32
	Submitted       time.Time
}

// Jobs returns information about all print jobs on this printer
func (p *Printer) Jobs() ([]JobInfo, error) {
	var bytesNeeded, jobsReturned uint32
	buf := make([]byte, 1)
	for {
		err := EnumJobs(p.h, 0, 255, 1, &buf[0], uint32(len(buf)), &bytesNeeded, &jobsReturned)
		if err == nil {
			break
		}
		if err != syscall.ERROR_INSUFFICIENT_BUFFER {
			return nil, err
		}
		if bytesNeeded <= uint32(len(buf)) {
			return nil, err
		}
		buf = make([]byte, bytesNeeded)
	}
	if jobsReturned <= 0 {
		return nil, nil
	}
	pjs := make([]JobInfo, 0, jobsReturned)
	ji := (*[2048]JOB_INFO_1)(unsafe.Pointer(&buf[0]))[:jobsReturned:jobsReturned]
	for _, j := range ji {
		pji := JobInfo{
			JobID:        j.JobID,
			StatusCode:   j.StatusCode,
			Priority:     j.Priority,
			Position:     j.Position,
			TotalPages:   j.TotalPages,
			PagesPrinted: j.PagesPrinted,
		}
		if j.MachineName != nil {
			pji.UserMachineName = windows.UTF16PtrToString(j.MachineName)
		}
		if j.UserName != nil {
			pji.UserName = windows.UTF16PtrToString(j.UserName)
		}
		if j.Document != nil {
			pji.DocumentName = windows.UTF16PtrToString(j.Document)
		}
		if j.DataType != nil {
			pji.DataType = windows.UTF16PtrToString(j.DataType)
		}
		if j.Status != nil {
			pji.Status = windows.UTF16PtrToString(j.Status)
		}
		if strings.TrimSpace(pji.Status) == "" {
			if pji.StatusCode == 0 {
				pji.Status += "Queue Paused, "
			}
			if pji.StatusCode&JOB_STATUS_PRINTING != 0 {
				pji.Status += "Printing, "
			}
			if pji.StatusCode&JOB_STATUS_PAUSED != 0 {
				pji.Status += "Paused, "
			}
			if pji.StatusCode&JOB_STATUS_ERROR != 0 {
				pji.Status += "Error, "
			}
			if pji.StatusCode&JOB_STATUS_DELETING != 0 {
				pji.Status += "Deleting, "
			}
			if pji.StatusCode&JOB_STATUS_SPOOLING != 0 {
				pji.Status += "Spooling, "
			}
			if pji.StatusCode&JOB_STATUS_OFFLINE != 0 {
				pji.Status += "Printer Offline, "
			}
			if pji.StatusCode&JOB_STATUS_PAPEROUT != 0 {
				pji.Status += "Out of Paper, "
			}
			if pji.StatusCode&JOB_STATUS_PRINTED != 0 {
				pji.Status += "Printed, "
			}
			if pji.StatusCode&JOB_STATUS_DELETED != 0 {
				pji.Status += "Deleted, "
			}
			if pji.StatusCode&JOB_STATUS_BLOCKED_DEVQ != 0 {
				pji.Status += "Driver Error, "
			}
			if pji.StatusCode&JOB_STATUS_USER_INTERVENTION != 0 {
				pji.Status += "User Action Required, "
			}
			if pji.StatusCode&JOB_STATUS_RESTART != 0 {
				pji.Status += "Restarted, "
			}
			if pji.StatusCode&JOB_STATUS_COMPLETE != 0 {
				pji.Status += "Sent to Printer, "
			}
			if pji.StatusCode&JOB_STATUS_RETAINED != 0 {
				pji.Status += "Retained, "
			}
			if pji.StatusCode&JOB_STATUS_RENDERING_LOCALLY != 0 {
				pji.Status += "Rendering on Client, "
			}
			pji.Status = strings.TrimRight(pji.Status, ", ")
		}
		pji.Submitted = time.Date(
			int(j.Submitted.Year),
			time.Month(int(j.Submitted.Month)),
			int(j.Submitted.Day),
			int(j.Submitted.Hour),
			int(j.Submitted.Minute),
			int(j.Submitted.Second),
			int(1000*j.Submitted.Milliseconds),
			time.Local,
		).UTC()
		pjs = append(pjs, pji)
	}
	return pjs, nil
}

// DriverInfo returns information about printer p driver.
func (p *Printer) DriverInfo() (*DriverInfo, error) {
	var needed uint32
	b := make([]byte, 1024*10)
	for {
		err := GetPrinterDriver(p.h, nil, 8, &b[0], uint32(len(b)), &needed)
		if err == nil {
			break
		}
		if err != syscall.ERROR_INSUFFICIENT_BUFFER {
			return nil, err
		}
		if needed <= uint32(len(b)) {
			return nil, err
		}
		b = make([]byte, needed)
	}
	di := (*DRIVER_INFO_8)(unsafe.Pointer(&b[0]))
	return &DriverInfo{
		Attributes:  di.PrinterDriverAttributes,
		Name:        windows.UTF16PtrToString(di.Name),
		DriverPath:  windows.UTF16PtrToString(di.DriverPath),
		Environment: windows.UTF16PtrToString(di.Environment),
	}, nil
}

func (p *Printer) StartDocument(name, datatype string) error {
	d := DOC_INFO_1{
		DocName:    &(syscall.StringToUTF16(name))[0],
		OutputFile: nil,
		Datatype:   &(syscall.StringToUTF16(datatype))[0],
	}
	return StartDocPrinter(p.h, 1, &d)
}

// StartRawDocument calls StartDocument and passes either "RAW" or "XPS_PASS"
// as a document type, depending if printer driver is XPS-based or not.
func (p *Printer) StartRawDocument(name string) error {
	di, err := p.DriverInfo()
	if err != nil {
		return err
	}
	// See https://support.microsoft.com/en-us/help/2779300/v4-print-drivers-using-raw-mode-to-send-pcl-postscript-directly-to-the
	// for details.
	datatype := "RAW"
	if di.Attributes&PRINTER_DRIVER_XPS != 0 {
		datatype = "XPS_PASS"
	}
	return p.StartDocument(name, datatype)
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
