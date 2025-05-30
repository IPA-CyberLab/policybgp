package asinfo

import (
	"compress/gzip"
	"encoding/csv"
	"fmt"
	"io"
	"net/netip"
	"os"
	"strconv"
	"strings"

	"go.uber.org/zap"
)

// ASInfo represents information about an Autonomous System and its IP ranges
type ASInfo struct {
	Organization string         // Organization name
	Prefixes     []netip.Prefix // List of IP prefixes (CIDR blocks) owned by this AS
}

// ASInfoMap maps ASN numbers to ASInfo structs
type ASInfoMap map[int]*ASInfo

func ParseASInfoCSV(r io.Reader, l *zap.Logger) (ASInfoMap, error) {
	s := l.Named("asinfo.ParseASInfoCSV").Sugar()

	csvReader := csv.NewReader(r)

	asn := make(ASInfoMap)

	lineNumber := 0
	s.Debug("Starting to parse CSV lines")
	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("error reading CSV at line %d: %w", lineNumber, err)
		}

		lineNumber++
		if lineNumber%100000 == 0 {
			s.Infof("Parsed %d lines so far", lineNumber)
		}

		s := s.With("line", lineNumber)

		if len(record) != 4 {
			return nil, fmt.Errorf("invalid record format at line %d: expected 4 fields, got %d", lineNumber, len(record))
		}

		startIP, err := netip.ParseAddr(record[0])
		if err != nil {
			return nil, fmt.Errorf("invalid start IP %q at line %d: %w", record[0], lineNumber, err)
		}

		endIP, err := netip.ParseAddr(record[1])
		if err != nil {
			return nil, fmt.Errorf("invalid end IP %q at line %d: %w", record[1], lineNumber, err)
		}

		asnNumber, err := strconv.Atoi(record[2])
		if err != nil {
			return nil, fmt.Errorf("invalid ASN %q at line %d: %w", record[2], lineNumber, err)
		}

		// Remove quotes from organization name if present
		orgName := strings.Trim(record[3], "\"")

		prefixes, err := ipRangeToCIDRs(startIP, endIP)
		if err != nil {
			return nil, fmt.Errorf("error converting IP range to CIDRs at line %d: %w", lineNumber, err)
		}

		info := asn[asnNumber]
		if info == nil {
			info = &ASInfo{
				Organization: orgName,
				Prefixes:     make([]netip.Prefix, 0, len(prefixes)),
			}
			asn[asnNumber] = info
		} else if info.Organization != orgName {
			s.Warnf("ASN %d has multiple organizations: %q and %q",
				asnNumber, info.Organization, orgName)
		}
		info.Prefixes = append(info.Prefixes, prefixes...)
	}

	s.Infow("Finished parsing ASN database",
		"total_asns", len(asn),
		"total_lines", lineNumber)
	return asn, nil
}

func ParseASInfoCSVFromFile(path string, l *zap.Logger) (ASInfoMap, error) {
	s := l.Named("asinfo.ParseASInfoCSVFromFile").Sugar().With("path", path)

	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("error opening file %q: %w", path, err)
	}
	defer f.Close()

	// Peek the first 2 bytes to check for gzip magic number
	peekBuf := make([]byte, 2)
	n, err := f.Read(peekBuf)
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("error reading file header: %w", err)
	}

	// Reset file position to beginning
	if _, err := f.Seek(0, 0); err != nil {
		return nil, fmt.Errorf("error seeking to file start: %w", err)
	}

	var reader io.Reader = f

	// Check if file is gzipped (magic bytes: 0x1f, 0x8b)
	if n >= 2 && peekBuf[0] == 0x1f && peekBuf[1] == 0x8b {
		s.Debug("detected gzipped file, using gzip reader")
		gzReader, err := gzip.NewReader(f)
		if err != nil {
			return nil, fmt.Errorf("error creating gzip reader: %w", err)
		}
		defer gzReader.Close()
		reader = gzReader
	}

	return ParseASInfoCSV(reader, l)
}
