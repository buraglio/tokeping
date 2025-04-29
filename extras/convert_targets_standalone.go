package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

type Entry struct {
	Name   string
	Probe  string
	Host   string
	Lookup string
	Server string
}

// parse reads a SmokePing targets-format input and returns entries
func parse(r io.Reader) []Entry {
	scanner := bufio.NewScanner(r)
	var entries []Entry
	var curr *Entry
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		// New entry starts with two or more pluses
		if strings.HasPrefix(trimmed, "++") {
			if curr != nil {
				entries = append(entries, *curr)
			}
			curr = &Entry{}
			continue
		}
		if curr == nil {
			continue
		}
		// parse lines like "key = value"
		if !strings.Contains(trimmed, "=") {
			continue
		}
		parts := strings.SplitN(trimmed, "=", 2)
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		switch key {
		case "menu":
			curr.Name = val
		case "probe":
			curr.Probe = val
		case "host":
			curr.Host = val
		case "lookup":
			curr.Lookup = val
		case "server":
			curr.Server = val
		}
	}
	if curr != nil {
		entries = append(entries, *curr)
	}
	return entries
}

func main() {
	infile := "Targets.txt"
	outfile := "config.yaml"

	// Open input
	f, err := os.Open(infile)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error opening Targets.txt:", err)
		os.Exit(1)
	}
	defer f.Close()

	entries := parse(f)

	// Create output file
	out, err := os.Create(outfile)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error creating config.yaml:", err)
		os.Exit(1)
	}
	defer out.Close()

	// Write YAML
	fmt.Fprintln(out, "probes:")
	for _, e := range entries {
		typ := "ping"
		target := e.Host
		if strings.EqualFold(e.Probe, "DNS") {
			typ = "dns"
			if e.Lookup != "" {
				target = e.Lookup
			}
		}
		fmt.Fprintf(out, "  - name: \"%s\"\n", e.Name)
		fmt.Fprintf(out, "    type: %s\n", typ)
		fmt.Fprintf(out, "    target: \"%s\"\n", target)
		fmt.Fprintln(out, "    interval: 5s")
		if typ == "dns" {
			resolver := e.Server
			if resolver == "" {
				resolver = e.Host
			}
			if !strings.Contains(resolver, ":") {
				resolver = resolver + ":53"
			}
			fmt.Fprintf(out, "    resolver: \"%s\"\n", resolver)
		}
	}

	fmt.Println("Generated", outfile)
}
