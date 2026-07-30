// Command unmask detects homograph / confusable look-alikes by the UTS-39
// skeleton: it reduces each label to the canonical form under which every visual
// look-alike collides, and JOINs that skeleton against a brand list.
//
//	unmask skeleton [labels...]                        # skeleton + scripts + mixed-script per label (stdin if no args)
//	unmask check -t <brands-file> [-all] [labels...]   # JOIN labels against a brand list; report confusable / mixed-script hits
//	unmask version
//
// Brands (check -t) and stdin labels are one per line. Labels are the command
// arguments, or one per line on stdin when none are given. The caller supplies the
// U-label (pre-punycode Unicode host, e.g. idna.ToUnicode) normalised to NFC; unmask
// does not decode punycode or normalise — it only skeletonises (and lower-cases).
//
// `check` reports the actionable hit per label: a confusable brand JOIN when one
// exists (label <tab> brand <tab> confusable), else the target-independent
// mixed-script signal when the label mixes scripts (label <tab> - <tab>
// mixed-script). A label with neither is skipped unless -all is set, which marks it
// "-". The confusable row is preferred because it names the brand; a mixed-script
// label absent from the brand list is still surfaced, so no signal is lost.
package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/netstar-labs/unmask"
)

// stamped by the build via -ldflags -X.
var (
	version = "dev"
	build   = "none"
)

func main() {
	if len(os.Args) < 2 {
		usage()
	}
	var err error
	switch os.Args[1] {
	case "skeleton":
		err = skeleton(os.Args[2:])
	case "check":
		err = check(os.Args[2:])
	case "version", "-version", "--version", "-v":
		// Unicode() is the release the pinned tables were generated from — the
		// skeleton is a clustering index, not a lookup key, so it is informational,
		// not a migration stamp (see the package doc).
		fmt.Printf("unmask %s (%s), Unicode %s\n", version, build, unmask.Unicode())
	default:
		usage()
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "unmask:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: unmask <skeleton|check|version> [flags] [labels...]")
	fmt.Fprintln(os.Stderr, "  unmask skeleton [labels...]                skeleton + scripts + mixed per label (stdin if no args)")
	fmt.Fprintln(os.Stderr, "  unmask check -t <brands-file> [labels...]  JOIN labels against a brand list; confusable / mixed-script hits")
	os.Exit(2)
}

// skeleton prints "label <tab> skeleton <tab> scripts <tab> mixed" for each label,
// taken from the arguments or, when there are none, one per line from stdin. The
// scripts column is the distinct non-neutral scripts joined by ",", or "-" when the
// label is entirely script-neutral (all digits/punctuation). This is the raw
// per-label [unmask.Analyze] view — a diagnostic, not a verdict: the skeleton is a
// clustering key that only means something JOINed against a target list (see check).
func skeleton(args []string) error {
	w := bufio.NewWriter(os.Stdout)
	defer w.Flush()
	emit := func(label string) {
		if label == "" {
			return
		}
		r := unmask.Analyze(label)
		scripts := "-"
		if len(r.Scripts) > 0 {
			scripts = strings.Join(r.Scripts, ",")
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%v\n", label, r.Skeleton, scripts, r.MixedScript)
	}

	if len(args) > 0 {
		for _, label := range args {
			emit(label)
		}
		return nil
	}
	// No label arguments: read one label per line from stdin.
	sc := newLineScanner(os.Stdin)
	for sc.Scan() {
		emit(strings.TrimSpace(sc.Text()))
	}
	return sc.Err()
}

// check JOINs each label against the brand list in -t and reports the actionable
// hit. It is the JOIN that is the actual product — [unmask.Confusable] carries the
// not-equal guard, so the legitimate brand queried against itself is not a hit, and
// the skeleton is never used as a bare lookup key. Each hit prints as: label <tab>
// brand <tab> kind, where kind is "confusable" (a brand JOIN) or "mixed-script" (the
// target-independent signal, when no brand matched). A label with neither is skipped
// unless -all is set, which marks it "-".
func check(args []string) error {
	fs := flag.NewFlagSet("check", flag.ExitOnError)
	tfile := fs.String("t", "", "brand target list, newline-delimited (required)")
	all := fs.Bool("all", false, "also print labels with no hit, marked with -")
	fs.Parse(args)

	if *tfile == "" {
		return errors.New("check: -t <brands-file> is required")
	}
	brands, err := readLines(*tfile)
	if err != nil {
		return err
	}

	w := bufio.NewWriter(os.Stdout)
	defer w.Flush()
	emit := func(label string) {
		if label == "" {
			return
		}
		// Confusable brand JOIN first: it names the brand and is the stronger,
		// actionable signal. First match wins — a label collides with two brands
		// only if those brands share a skeleton, which is rare and equivalent.
		for _, b := range brands {
			if unmask.Confusable(label, b) {
				fmt.Fprintf(w, "%s\t%s\tconfusable\n", label, b)
				return
			}
		}
		// No brand JOIN: the mixed-script signal is independent of the target list,
		// so a suspicious label absent from the brand list is still surfaced here.
		if unmask.MixedScript(label) {
			fmt.Fprintf(w, "%s\t-\tmixed-script\n", label)
			return
		}
		if *all {
			fmt.Fprintf(w, "%s\t-\t-\n", label)
		}
	}

	if labels := fs.Args(); len(labels) > 0 {
		for _, label := range labels {
			emit(label)
		}
		return nil
	}
	// No label arguments: read one label per line from stdin.
	sc := newLineScanner(os.Stdin)
	for sc.Scan() {
		emit(strings.TrimSpace(sc.Text()))
	}
	return sc.Err()
}

// readLines reads path as newline-delimited entries, trimming surrounding
// whitespace and skipping blank lines.
func readLines(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var out []string
	sc := newLineScanner(f)
	for sc.Scan() {
		if line := strings.TrimSpace(sc.Text()); line != "" {
			out = append(out, line)
		}
	}
	return out, sc.Err()
}

// newLineScanner returns a bufio.Scanner over r that tolerates lines up to 1 MiB.
func newLineScanner(r io.Reader) *bufio.Scanner {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 1<<20)
	return sc
}
