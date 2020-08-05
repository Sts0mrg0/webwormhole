// Command ww moves files and other data over WebRTC.
package main

import (
	crand "crypto/rand"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"

	"rsc.io/qr"
	"webwormhole.io/wordlist"
	"webwormhole.io/wormhole"
)

var subcmds = map[string]func(args ...string){
	"send":    send,
	"receive": receive,
	"pipe":    pipe,
	"server":  server,
}

var (
	verbose = flag.Bool("verbose", false, "verbose logging")
	sigserv = flag.String("signal", "https://wrmhl.link/", "signalling server to use")
)

var stderr = flag.CommandLine.Output()

func usage() {
	fmt.Fprintf(stderr, "webwormhole creates ephemeral pipes between computers.\n\n")
	fmt.Fprintf(stderr, "usage:\n\n")
	fmt.Fprintf(stderr, "  %s [flags] <command> [arguments]\n\n", os.Args[0])
	fmt.Fprintf(stderr, "commands:\n")
	for key := range subcmds {
		fmt.Fprintf(stderr, "  %s\n", key)
	}
	fmt.Fprintf(stderr, "\nflags:\n")
	flag.PrintDefaults()
}

func main() {
	flag.Usage = usage
	flag.Parse()
	if flag.NArg() < 1 {
		usage()
		os.Exit(2)
	}
	if *verbose {
		wormhole.Verbose = true
	}
	cmd, ok := subcmds[flag.Arg(0)]
	if !ok {
		flag.Usage()
		os.Exit(2)
	}
	cmd(flag.Args()...)
}

func fatalf(format string, v ...interface{}) {
	fmt.Fprintf(stderr, format+"\n", v...)
	os.Exit(1)
}

func newConn(code string, length int) *wormhole.DataChannel {
	var w *wormhole.Wormhole
	var err error
	var passbytes []byte
	if code == "" {
		// New wormhole.
		passbytes = make([]byte, length)
		if _, err := io.ReadFull(crand.Reader, passbytes); err != nil {
			fatalf("could not generate password: %v", err)
		}
		w, err = wormhole.New(*sigserv)
		if err == wormhole.ErrBadVersion {
			fatalf("%s%s%s",
				"the signalling server is running an incompatable version.\n",
				"try upgrading the client:\n\n",
				"    go get webwormhole.io/cmd/ww\n")
		}
		if err != nil {
			fatalf("could not dial: %v", err)
		}
		printcode(w.Slot + "-" + strings.Join(wordlist.Encode(passbytes), "-"))
	} else {
		// Join wormhole.
		parts := strings.Split(code, "-")
		passbytes, _ = wordlist.Decode(parts[1:])
		if passbytes == nil {
			fatalf("could not decode password")
		}
		w, err = wormhole.Join(*sigserv, parts[0])
		if err == wormhole.ErrBadVersion {
			fatalf("%s%s%s",
				"the signalling server is running an incompatable version.\n",
				"try upgrading the client:\n\n",
				"    go get webwormhole.io/cmd/ww\n")
		}
		if err != nil {
			fatalf("could not dial: %v", err)
		}
	}

	c, err := w.DialDataChannel(string(passbytes))
	if err != nil {
		fatalf("could not dial: %v", err)
	}
	if w.IsRelay() {
		fmt.Fprintf(stderr, "connected: relay\n")
	} else {
		fmt.Fprintf(stderr, "connected: direct\n")
	}
	return c
}

func printcode(code string) {
	fmt.Fprintf(stderr, "%s\n", code)
	u, err := url.Parse(*sigserv)
	if err != nil {
		return
	}
	u.Fragment = code
	qrcode, err := qr.Encode(u.String(), qr.L)
	if err != nil {
		return
	}
	for x := 0; x < qrcode.Size; x++ {
		fmt.Fprintf(stderr, "█")
	}
	fmt.Fprintf(stderr, "████████\n")
	for x := 0; x < qrcode.Size; x++ {
		fmt.Fprintf(stderr, "█")
	}
	fmt.Fprintf(stderr, "████████\n")
	for y := 0; y < qrcode.Size; y += 2 {
		fmt.Fprintf(stderr, "████")
		for x := 0; x < qrcode.Size; x++ {
			switch {
			case qrcode.Black(x, y) && qrcode.Black(x, y+1):
				fmt.Fprintf(stderr, " ")
			case qrcode.Black(x, y):
				fmt.Fprintf(stderr, "▄")
			case qrcode.Black(x, y+1):
				fmt.Fprintf(stderr, "▀")
			default:
				fmt.Fprintf(stderr, "█")
			}
		}
		fmt.Fprintf(stderr, "████\n")
	}
	for x := 0; x < qrcode.Size; x++ {
		fmt.Fprintf(stderr, "█")
	}
	fmt.Fprintf(stderr, "████████\n")
	for x := 0; x < qrcode.Size; x++ {
		fmt.Fprintf(stderr, "█")
	}
	fmt.Fprintf(stderr, "████████\n")
	fmt.Fprintf(stderr, "%s\n", u.String())
}
