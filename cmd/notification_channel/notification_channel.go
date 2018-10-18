// +build windows

package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/jazzy-crane/printer"
)

var jobNotify = []uint16{
	printer.JOB_NOTIFY_FIELD_PRINTER_NAME,
	printer.JOB_NOTIFY_FIELD_MACHINE_NAME,
	printer.JOB_NOTIFY_FIELD_PORT_NAME,
	printer.JOB_NOTIFY_FIELD_USER_NAME,
	printer.JOB_NOTIFY_FIELD_STATUS,
	printer.JOB_NOTIFY_FIELD_SUBMITTED,
	printer.JOB_NOTIFY_FIELD_DOCUMENT,
	printer.JOB_NOTIFY_FIELD_DATATYPE,
	printer.JOB_NOTIFY_FIELD_PRINT_PROCESSOR,
	printer.JOB_NOTIFY_FIELD_PAGES_PRINTED,
	printer.JOB_NOTIFY_FIELD_BYTES_PRINTED,
}

func main() {
	multiplexed := make(chan *printer.NotifyInfo)
	done := make(chan struct{})

	go func() {
		for notification := range multiplexed {
			fmt.Printf("\n%s\n", notification)
		}
	}()

	notifyOptions := &printer.PRINTER_NOTIFY_OPTIONS{
		Version: 2,
		Flags:   0,
		Count:   1,
		PTypes: &printer.PRINTER_NOTIFY_OPTIONS_TYPE{
			Type:    uint16(printer.JOB_NOTIFY_TYPE),
			Count:   uint32(len(jobNotify)),
			PFields: &jobNotify[0],
		},
	}

outerloop:
	for {
		wg := sync.WaitGroup{}
		pnames, err := printer.ReadNames()
		if err != nil {
			fmt.Println("printer.ReadNames", err)
			time.Sleep(10 * time.Second)
			continue
		}

		for _, pname := range pnames {
			p, err := printer.Open(pname)
			if err != nil {
				fmt.Println("printer.Open", pname, err)
				time.Sleep(10 * time.Second)
				goto outerloop
			}

			n, err := p.GetNotifications(done, printer.PRINTER_CHANGE_ALL, 0, notifyOptions)
			if err != nil {
				fmt.Println("printer.GetNotifications", pname, err)
				time.Sleep(10 * time.Second)
				goto outerloop
			}

			wg.Add(1)

			go func(notifications <-chan *printer.NotifyInfo) {
				defer wg.Done()
				fmt.Println("Starting notification goroutine")
				for n := range notifications {
					multiplexed <- n
				}
				fmt.Println("Notification goroutine returned")
			}(n)
		}

		wg.Wait()
		fmt.Println("All notification goroutines returned, probably due to spooler service stop/restart")
	}
}
