// +build windows
package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jazzy-crane/printer"
)

func main() {
	pnames, err := printer.ReadNames()
	if err != nil {
		log.Println("Error", err)
		os.Exit(1)
	}

	printers := make([]*printer.Printer, len(pnames))
	for i, pname := range pnames {
		var err error
		printers[i], err = printer.Open(pname)
		if err != nil {
			log.Println("Error printer.Open", pname, err)
			os.Exit(1)
		}
	}

	multiplexed := make(chan printer.PrinterNotifyInfo)

	for _, p := range printers {
		notifications, err := p.StartChangeNotifications(printer.JOB_NOTIFY_TYPE, printer.JobNotifyAll)
		if err != nil {
			log.Println("Error StartChangeNotifications", err)
			os.Exit(1)
		}

		go func(p <-chan printer.PrinterNotifyInfo) {
			for msg := range p {
				multiplexed <- msg
			}
		}(notifications)
	}

	timeout := time.After(time.Minute)

	running := true
	for running {
		select {
		case <-timeout:
			running = false
		case pni := <-multiplexed:
			fmt.Printf("\nNew print notification (cause 0x%X)\n", pni.Cause)
			for _, item := range pni.Data {
				fmt.Println(item)
			}
		}
	}

	for _, p := range printers {
		if err := p.EndChangeNotifications(); err != nil {
			log.Println("Error EndChangeNotifications", err)
			os.Exit(1)
		}
		if err := p.Close(); err != nil {
			log.Println("Error Close", err)
			os.Exit(1)
		}
	}
}
