// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build windows

// print command prints text documents to selected printer.
package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"

	"github.com/alexbrainman/printer"
)

var (
	copies    = flag.Int("n", 1, "number of copies to print")
	printerId = flag.String("p", findDefaultPrinter(), "printer name or printer index from printer list")
	doList    = flag.Bool("l", false, "list printers")
)

func findDefaultPrinter() string {
	p, err := printer.Default()
	if err != nil {
		return ""
	}
	return p
}

func listPrinters() error {
	printers, err := printer.ReadNames()
	if err != nil {
		return err
	}
	defaultPrinter, err := printer.Default()
	if err != nil {
		return err
	}
	for i, p := range printers {
		s := " "
		if p == defaultPrinter {
			s = "*"
		}
		fmt.Printf(" %s %d. %s\n", s, i, p)
	}
	return nil
}

func selectPrinter() (string, error) {
	n, err := strconv.Atoi(*printerId)
	if err != nil {
		// must be a printer name
		return *printerId, nil
	}
	printers, err := printer.ReadNames()
	if err != nil {
		return "", err
	}
	if n < 0 {
		return "", fmt.Errorf("printer index (%d) cannot be negative", n)
	}
	if n >= len(printers) {
		return "", fmt.Errorf("printer index (%d) is too large, there are only %d printers", n, len(printers))
	}
	return printers[n], nil
}

func printOneDocument(printerName, documentName string, lines []string) error {
	p, err := printer.Open(printerName)
	if err != nil {
		return err
	}
	defer p.Close()

	err = p.StartRawDocument(documentName)
	if err != nil {
		return err
	}
	defer p.EndDocument()

	err = p.StartPage()
	if err != nil {
		return err
	}

	for _, line := range lines {
		fmt.Fprintf(p, "%s\r\n", line)
	}

	return p.EndPage()
}

func printDocument(path string) error {
	if *copies < 0 {
		return fmt.Errorf("number of copies to print (%d) cannot be negative", *copies)
	}

	output, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	lines := strings.Split(string(output), "\n")

	printerName, err := selectPrinter()
	if err != nil {
		return err
	}

	for i := 0; i < *copies; i++ {
		err := printOneDocument(printerName, path, lines)
		if err != nil {
			return err
		}
	}
	return nil
}

func usage() {
	fmt.Fprintln(os.Stderr)
	fmt.Fprintf(os.Stderr, "usage: print [-n=<copies>] [-p=<printer>] <file-path-to-print>\n")
	fmt.Fprintf(os.Stderr, "       or\n")
	fmt.Fprintf(os.Stderr, "       print -l\n")
	fmt.Fprintln(os.Stderr)
	flag.PrintDefaults()
	os.Exit(1)
}

func exit(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func main() {
	flag.Usage = usage
	flag.Parse()
	if *doList {
		exit(listPrinters())
	}
	switch len(flag.Args()) {
	case 0:
		fmt.Fprintf(os.Stderr, "no document path to print provided\n")
	case 1:
		exit(printDocument(flag.Arg(0)))
	default:
		fmt.Fprintf(os.Stderr, "too many parameters provided\n")
	}
	usage()
}
