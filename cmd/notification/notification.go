// +build windows

package main

import (
	"fmt"
	"os"
	"time"

	"github.com/jazzy-crane/printer"
)

func notifyInfoHasStatus(ni *printer.NotifyInfo, status uint32) bool {
	for _, d := range ni.Data {
		if d.Field == printer.JOB_NOTIFY_FIELD_STATUS && ((d.Value.(uint32) & status) == status) {
			return true
		}
	}

	return false
}

func main() {
	pnames, err := printer.ReadNames()
	if err != nil {
		fmt.Println("printer.ReadNames", err)
		os.Exit(1)
	}

	notifyOptions := &printer.PRINTER_NOTIFY_OPTIONS{
		Version: 2,
		Flags:   0,
		Count:   1,
		PTypes: &printer.PRINTER_NOTIFY_OPTIONS_TYPE{
			Type:    uint16(printer.JOB_NOTIFY_TYPE),
			Count:   uint32(len(printer.JobNotifyAll)),
			PFields: &printer.JobNotifyAll[0],
		},
	}

	for _, pname := range pnames {
		p, err := printer.Open(pname)
		if err != nil {
			fmt.Println("printer.Open", pname, err)
			os.Exit(1)
		}

		notifications, err := p.ChangeNotifications(printer.PRINTER_CHANGE_ALL, 0, notifyOptions)
		if err != nil {
			fmt.Println("printerChangeNotifications", err)
			os.Exit(1)
		}

		go func(p *printer.Printer, pcnh *printer.ChangeNotificationHandle) {
			defer pcnh.Close()
			defer p.Close()

			for {
				pni, err := pcnh.Next(nil)
				if err != nil {
					if err != printer.ErrNoNotification {
						fmt.Println("ChangeNotificationHandle::Next", err)
					}
					continue
				}

				if (pni.Cause & printer.PRINTER_CHANGE_ADD_JOB) != 0 {
					fmt.Println("\nAdded job", pni.Data[0].ID)
					job, err := p.Job(pni.Data[0].ID)
					if err != nil {
						fmt.Println("p.Job", err)
					} else {
						fmt.Printf("%#v\n", job)
					}

					err = p.SetJob(pni.Data[0].ID, nil, printer.JOB_CONTROL_RETAIN)
					if err != nil {
						fmt.Println("p.SetJob", err)
					} else {
						fmt.Println("JOB_CONTROL_RETAIN success")
					}
				}

				if notifyInfoHasStatus(pni, printer.JOB_STATUS_COMPLETE|printer.JOB_STATUS_RETAINED) {
					fmt.Println("\nRetained job done", pni.Data[0].ID)
					job, err := p.Job(pni.Data[0].ID)
					if err != nil {
						fmt.Println("p.Job", err)
					} else {
						fmt.Printf("%#v\n", job)
					}

					jobs, err := p.Jobs()
					if err != nil {
						fmt.Println("p.Jobs", err)
					} else {
						for i, j := range jobs {
							fmt.Printf("%d: %#v\n", i, j)
						}
					}

					time.Sleep(time.Second * 10)

					err = p.SetJob(pni.Data[0].ID, nil, printer.JOB_CONTROL_RELEASE)
					if err != nil {
						fmt.Println("p.SetJob", err)
					} else {
						fmt.Println("JOB_CONTROL_RELEASE success")
					}
				}
			}
		}(p, notifications)
	}

	for {
		time.Sleep(time.Second)
	}
}
