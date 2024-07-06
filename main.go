package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"sort"
	"strings"
)

type Flags struct {
	inputFile      string
	outputPrivate  string
	outputPublic   string
	outputLoopback string
	ipv4           bool
	ipv6           bool
}

func (f *Flags) Validate() error {
	in, err := os.Stat(f.inputFile)
	if err != nil {
		return fmt.Errorf("failed to stat input file: %v", err)
	}

	if in.IsDir() {
		return fmt.Errorf("input file is a directory")
	}

	if allEmptyStrings(f.outputPrivate, f.outputPublic, f.outputLoopback) {
		return fmt.Errorf("no output files specified")
	}

	if f.outputPrivate != "" {
		_, err := os.Stat(f.outputPrivate)
		if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("output file %q already exists", f.outputPrivate)
		}
	}

	if f.outputPublic != "" {
		_, err := os.Stat(f.outputPublic)
		if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("output file %q already exists", f.outputPublic)
		}
	}

	if f.outputLoopback != "" {
		_, err := os.Stat(f.outputLoopback)
		if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("output file %q already exists", f.outputLoopback)
		}
	}

	if !f.ipv4 && !f.ipv6 {
		return fmt.Errorf("no ip version specified")
	}

	return nil
}

func allEmptyStrings(first string, others ...string) bool {
	if first != "" {
		return false
	}

	for _, s := range others {
		if s != "" {
			return false
		}
	}

	return true
}

type ipSubMap struct {
	private  fragment
	public   fragment
	loopback fragment

	ipv4 bool
	ipv6 bool
}

type fragment struct {
	out io.Writer
	m   map[string][]string
}

func (f *fragment) append(ip string, subdomain string) {
	if f.m == nil {
		return
	}
	f.m[ip] = append(f.m[ip], subdomain)
}

func (f *fragment) write() error {
	if f.m == nil || f.out == nil {
		return nil
	}
	keys := make([]string, 0, len(f.m))
	for k := range f.m {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	for _, k := range keys {
		output := fmt.Sprintf("%s %s\n", k, strings.Join(f.m[k], ","))
		if _, err := f.out.Write([]byte(output)); err != nil {
			return err
		}
	}

	return nil
}

func (m *ipSubMap) enumerate(in io.Reader) error {
	scanner := bufio.NewScanner(in)
	var errs []error
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		if err := m.resolve(line); err != nil {
			errs = append(errs, err)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to read input: %v", err)
	}

	return errors.Join(errs...)
}

func (m *ipSubMap) write() error {
	var errs []error
	if err := m.private.write(); err != nil {
		errs = append(errs, fmt.Errorf("failed to write private ip subdomains: %v", err))
	}

	if err := m.public.write(); err != nil {
		errs = append(errs, fmt.Errorf("failed to write public ip subdomains: %v", err))
	}

	if err := m.loopback.write(); err != nil {
		errs = append(errs, fmt.Errorf("failed to write loopback ip subdomains: %v", err))
	}

	return errors.Join(errs...)
}

func (m *ipSubMap) resolve(subdomain string) error {
	ips, err := net.LookupIP(subdomain)
	if err != nil {
		return fmt.Errorf("failed to resolve subdomain %q: %v", subdomain, err)
	}

	for _, ip := range ips {
		if ip.To4() == nil && !m.ipv6 {
			continue
		}
		if ip.To4() != nil && !m.ipv4 {
			continue
		}

		ipStr := ip.String()
		switch {
		case ip.IsLoopback():
			m.loopback.append(ipStr, subdomain)
		case ip.IsPrivate():
			m.private.append(ipStr, subdomain)
		default:
			m.public.append(ipStr, subdomain)
		}
	}

	return nil
}

func main() {
	logger := slog.New(
		slog.NewTextHandler(
			os.Stdout,
			&slog.HandlerOptions{
				AddSource: true,
				Level:     slog.LevelInfo,
			},
		),
	)

	logger = logger.With(slog.String("app", "ipsubmap"))
	var flags Flags

	flag.StringVar(&flags.inputFile, "file", "", "Input file")
	flag.StringVar(&flags.outputPrivate, "out-private", "", "Output file for private ip subdomains")
	flag.StringVar(&flags.outputPublic, "out-public", "", "Output file for public ip subdomains")
	flag.StringVar(&flags.outputLoopback, "out-loopback", "", "Output file for loopback ip subdomains")
	flag.BoolVar(&flags.ipv4, "ipv4", true, "Resolve ipv4 addresses. True by default")
	flag.BoolVar(&flags.ipv6, "ipv6", true, "Resolve ipv6 addresses. True by default")

	flag.Parse()

	if err := flags.Validate(); err != nil {
		logger.Error("failed to validate flags", "error", err)
		flag.Usage()
		os.Exit(1)
	}

	in, err := os.Open(flags.inputFile)
	if err != nil {
		logger.Error("failed to open input file", "error", err)
		os.Exit(1)
	}
	defer in.Close()

	buf := bufio.NewReader(in)

	mapper := &ipSubMap{
		ipv4: flags.ipv4,
		ipv6: flags.ipv6,
	}
	if flags.outputPrivate != "" {
		out, err := os.Create(flags.outputPrivate)
		if err != nil {
			logger.Error("failed to create output (private) file", "error", err)
			os.Exit(1)
		}
		defer out.Close()
		mapper.private = fragment{out: out, m: make(map[string][]string)}
	}

	if flags.outputPublic != "" {
		out, err := os.Create(flags.outputPublic)
		if err != nil {
			logger.Error("failed to create output (public) file", "error", err)
			os.Exit(1)
		}
		defer out.Close()
		mapper.public = fragment{out: out, m: make(map[string][]string)}
	}

	if flags.outputLoopback != "" {
		out, err := os.Create(flags.outputLoopback)
		if err != nil {
			logger.Error("failed to create output (loopback) file", "error", err)
			os.Exit(1)
		}
		defer out.Close()
		mapper.loopback = fragment{out: out, m: make(map[string][]string)}
	}

	if err := mapper.enumerate(buf); err != nil {
		logger.Error("Encountered errors while enumerating", "error", err)
	}
	logger.Info("Writing output files")

	if err := mapper.write(); err != nil {
		logger.Error("Encountered errors while writing", "error", err)
		os.Exit(1)
	}
}
