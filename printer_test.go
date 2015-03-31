// Copyright 2013 The Go Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package printer

import (
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
	t.Fatal("Default printed %q is not listed amongst printers returned by ReadNames %q", name, names)
}
