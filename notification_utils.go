package printer

import (
	"bytes"
	"fmt"
	"strings"
	"syscall"
)

// JobNotifyAll is a util providing a slice of all JOB_NOTIFY_FIELD_* values
var JobNotifyAll = []uint16{
	JOB_NOTIFY_FIELD_PRINTER_NAME,
	JOB_NOTIFY_FIELD_MACHINE_NAME,
	JOB_NOTIFY_FIELD_PORT_NAME,
	JOB_NOTIFY_FIELD_USER_NAME,
	JOB_NOTIFY_FIELD_NOTIFY_NAME,
	JOB_NOTIFY_FIELD_DATATYPE,
	JOB_NOTIFY_FIELD_PRINT_PROCESSOR,
	JOB_NOTIFY_FIELD_PARAMETERS,
	JOB_NOTIFY_FIELD_DRIVER_NAME,
	JOB_NOTIFY_FIELD_DEVMODE,
	JOB_NOTIFY_FIELD_STATUS,
	JOB_NOTIFY_FIELD_STATUS_STRING,
	JOB_NOTIFY_FIELD_SECURITY_DESCRIPTOR,
	JOB_NOTIFY_FIELD_DOCUMENT,
	JOB_NOTIFY_FIELD_PRIORITY,
	JOB_NOTIFY_FIELD_POSITION,
	JOB_NOTIFY_FIELD_SUBMITTED,
	JOB_NOTIFY_FIELD_START_TIME,
	JOB_NOTIFY_FIELD_UNTIL_TIME,
	JOB_NOTIFY_FIELD_TIME,
	JOB_NOTIFY_FIELD_TOTAL_PAGES,
	JOB_NOTIFY_FIELD_PAGES_PRINTED,
	JOB_NOTIFY_FIELD_TOTAL_BYTES,
	JOB_NOTIFY_FIELD_BYTES_PRINTED,
	JOB_NOTIFY_FIELD_REMOTE_JOB_ID,
}

// JobNotifyFieldToString maps all JOB_NOTIFY_FIELD_* values to a human readable string
func JobNotifyFieldToString(field uint16) string {
	switch field {
	case JOB_NOTIFY_FIELD_PRINTER_NAME:
		return "Printer name"
	case JOB_NOTIFY_FIELD_MACHINE_NAME:
		return "Machine name"
	case JOB_NOTIFY_FIELD_PORT_NAME:
		return "Port name"
	case JOB_NOTIFY_FIELD_USER_NAME:
		return "User name"
	case JOB_NOTIFY_FIELD_NOTIFY_NAME:
		return "Notify name"
	case JOB_NOTIFY_FIELD_DATATYPE:
		return "Datatype"
	case JOB_NOTIFY_FIELD_PRINT_PROCESSOR:
		return "Print processor"
	case JOB_NOTIFY_FIELD_PARAMETERS:
		return "Parameters"
	case JOB_NOTIFY_FIELD_DRIVER_NAME:
		return "Driver name"
	case JOB_NOTIFY_FIELD_DEVMODE:
		return "Devmode"
	case JOB_NOTIFY_FIELD_STATUS:
		return "Status"
	case JOB_NOTIFY_FIELD_STATUS_STRING:
		return "Status(string)"
	case JOB_NOTIFY_FIELD_SECURITY_DESCRIPTOR:
		return "Security descriptor"
	case JOB_NOTIFY_FIELD_DOCUMENT:
		return "Document"
	case JOB_NOTIFY_FIELD_PRIORITY:
		return "Priority"
	case JOB_NOTIFY_FIELD_POSITION:
		return "Position"
	case JOB_NOTIFY_FIELD_SUBMITTED:
		return "Submitted time"
	case JOB_NOTIFY_FIELD_START_TIME:
		return "Start time"
	case JOB_NOTIFY_FIELD_UNTIL_TIME:
		return "Until time"
	case JOB_NOTIFY_FIELD_TIME:
		return "Time since start"
	case JOB_NOTIFY_FIELD_TOTAL_PAGES:
		return "Total pages"
	case JOB_NOTIFY_FIELD_PAGES_PRINTED:
		return "Pages printed"
	case JOB_NOTIFY_FIELD_TOTAL_BYTES:
		return "Total bytes"
	case JOB_NOTIFY_FIELD_BYTES_PRINTED:
		return "Bytes printed"
	case JOB_NOTIFY_FIELD_REMOTE_JOB_ID:
		return "Remote job id"
	}

	return "<UNKNOWN>"
}

func (pnid *NotifyInfoData) String() string {
	if pnid.Type == JOB_NOTIFY_TYPE {
		if pnid.Field == JOB_NOTIFY_FIELD_STATUS {
			return fmt.Sprintf("Job #%d %s: %v (%s)", pnid.ID, JobNotifyFieldToString(pnid.Field), pnid.Value, jobStatusCodeToString(pnid.Value.(uint32)))
		} else {
			return fmt.Sprintf("Job #%d %s: %v", pnid.ID, JobNotifyFieldToString(pnid.Field), pnid.Value)
		}
	} else if pnid.Type == PRINTER_NOTIFY_TYPE {
		return fmt.Sprintf("Printer Field %d Value %v", pnid.Field, pnid.Value)
	}

	return fmt.Sprintf("%#v", pnid)
}

func (pni *NotifyInfo) String() string {
	var buf bytes.Buffer

	fmt.Fprintf(&buf, "NotifyInfo cause 0x%X\n", pni.Cause)
	for _, item := range pni.Data {
		fmt.Fprintf(&buf, "%s\n", item.String())
	}

	return strings.TrimRight(buf.String(), "\n")
}

// GetNotifications wraps the whole FindFirstPrinterChangeNotification, WaitForSingleObject,
// FindNextPrinterChangeNotification, FindClosePrinterChangeNotification process and vends notifications out of a channel
// To finish notifications and cleanup, close the passed in done channel
func (p *Printer) GetNotifications(done <-chan struct{}, filter uint32, options uint32, printerNotifyOptions *PRINTER_NOTIFY_OPTIONS) (<-chan *NotifyInfo, error) {
	notificationHandle, err := p.ChangeNotifications(filter, options, printerNotifyOptions)
	if err != nil {
		return nil, err
	}

	out := make(chan *NotifyInfo)

	go func() {
		defer func() {
			notificationHandle.Close()
			close(out)
		}()
		for {
			// Ideally this should be syscall.INFINITE, but need to keep waking up to check the done channel
			rtn, err := notificationHandle.Wait(500)
			if err != nil {
				continue
			}

			if rtn == syscall.WAIT_TIMEOUT {
				select {
				case <-done:
					return
				default:
					continue
				}
			}

			if rtn != syscall.WAIT_FAILED {
				pni, err := notificationHandle.Next(nil)
				if err != nil {
					continue
				}

				select {
				case out <- pni:
				case <-done:
					return
				}
			} else {
				return
			}
		}
	}()

	return out, nil
}
