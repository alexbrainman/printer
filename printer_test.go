// Copyright 2013 The Go Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package printer

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestPrinter(t *testing.T) {
	name, err := Default()
	if err != nil {
		t.Fatalf("Default failed: %v", err)
	}

	p, err := Open(name)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer p.Close()

	err = p.StartDocument("my document", "RAW")
	if err != nil {
		t.Fatalf("StartDocument failed: %v", err)
	}
	defer p.EndDocument()
	err = p.StartPage()
	if err != nil {
		t.Fatalf("StartPage failed: %v", err)
	}
	fmt.Fprintf(p, "Hello %q\n", name)
	err = p.EndPage()
	if err != nil {
		t.Fatalf("EndPage failed: %v", err)
	}
}

func TestReadNames(t *testing.T) {
	names, err := ReadNames()
	if err != nil {
		t.Fatalf("ReadNames failed: %v", err)
	}
	name, err := Default()
	if err != nil {
		t.Fatalf("Default failed: %v", err)
	}
	// make sure default printer is listed
	for _, v := range names {
		if v == name {
			return
		}
	}
	t.Fatalf("Default printed %q is not listed amongst printers returned by ReadNames %q", name, names)
}

func TestDriverInfo(t *testing.T) {
	name, err := Default()
	if err != nil {
		t.Fatalf("Default failed: %v", err)
	}

	p, err := Open(name)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer p.Close()

	di, err := p.DriverInfo()
	if err != nil {
		t.Fatalf("DriverInfo failed: %v", err)
	}
	t.Logf("%+v", di)
}

func TestJobs(t *testing.T) {
	names, err := ReadNames()
	if err != nil {
		t.Fatalf("ReadNames failed: %v", err)
	}
	for _, name := range names {
		t.Log("Printer Name:", name)
		p, err := Open(name)
		if err != nil {
			t.Fatalf("Open failed: %v", err)
		}
		defer p.Close()

		pj, err := p.Jobs()
		if err != nil {
			t.Fatalf("Jobs failed: %v", err)
		}
		if len(pj) > 0 {
			t.Log("Print Jobs:", len(pj))
			for _, j := range pj {
				b, err := json.MarshalIndent(j, "", "   ")
				if err == nil && len(b) > 0 {
					t.Log(string(b))
				}
			}
		}
	}
}
