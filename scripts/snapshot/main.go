// Command snapshot checks the snapshot against PROVENANCE.json, and brings it up to a newer GA commit. Run it from the
// repository root; it needs Go's standard library and, to sync, git. Nothing touches the network.
//
//	go run ./scripts/snapshot verify
//	go run ./scripts/snapshot sync <GA checkout> <source commit>
package main

import (
	"fmt"
	"os"
)

func main() {
	var summary string
	var err error
	switch {
	case len(os.Args) == 2 && os.Args[1] == "verify":
		summary, err = verify(".")
	case len(os.Args) == 4 && os.Args[1] == "sync":
		summary, err = syncSnapshot(os.Args[2], os.Args[3], ".")
	default:
		fmt.Fprintln(os.Stderr, "usage: snapshot verify | snapshot sync <GA checkout> <source commit>")
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "FAIL:", err)
		os.Exit(1)
	}
	fmt.Println(summary)
}
