package main

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"path"
	"strconv"
	"strings"
)

// cell is one cached cell value. Shared and inline strings are resolved into
// text; numbers and formula results keep Excel's cached <v> text.
type cell struct{ text string }

func (c cell) empty() bool { return strings.TrimSpace(c.text) == "" }

// sheet maps an A1-style reference such as "B6" to its cell.
type sheet map[string]cell

type workbook struct {
	zip    *zip.Reader
	sheets map[string]string // sheet name -> zip path of its worksheet XML
	shared []string
}

func openWorkbook(r io.ReaderAt, size int64) (*workbook, error) {
	zr, err := zip.NewReader(r, size)
	if err != nil {
		return nil, fmt.Errorf("not an xlsx file: %w", err)
	}
	wb := &workbook{zip: zr, sheets: map[string]string{}}

	var book struct {
		Pr struct {
			Date1904 string `xml:"date1904,attr"`
		} `xml:"workbookPr"`
		Sheets []struct {
			Name string `xml:"name,attr"`
			RID  string `xml:"http://schemas.openxmlformats.org/officeDocument/2006/relationships id,attr"`
		} `xml:"sheets>sheet"`
	}
	if err := wb.decode("xl/workbook.xml", &book); err != nil {
		return nil, err
	}
	if book.Pr.Date1904 == "1" || book.Pr.Date1904 == "true" {
		return nil, fmt.Errorf("workbook uses the 1904 date system; only the 1900 system is supported")
	}
	var rels struct {
		Rels []struct {
			ID     string `xml:"Id,attr"`
			Target string `xml:"Target,attr"`
		} `xml:"Relationship"`
	}
	if err := wb.decode("xl/_rels/workbook.xml.rels", &rels); err != nil {
		return nil, err
	}
	targets := map[string]string{}
	for _, r := range rels.Rels {
		if after, ok := strings.CutPrefix(r.Target, "/"); ok {
			targets[r.ID] = after
		} else {
			targets[r.ID] = path.Join("xl", r.Target)
		}
	}
	for _, s := range book.Sheets {
		wb.sheets[s.Name] = targets[s.RID]
	}

	var sst struct {
		Items []richText `xml:"si"`
	}
	if err := wb.decode("xl/sharedStrings.xml", &sst); err != nil && !isMissing(err) {
		return nil, err
	}
	for _, si := range sst.Items {
		wb.shared = append(wb.shared, si.String())
	}
	return wb, nil
}

// richText is an <si> or <is> element: either a plain <t> or formatted runs.
type richText struct {
	T    string `xml:"t"`
	Runs []struct {
		T string `xml:"t"`
	} `xml:"r"`
}

func (r richText) String() string {
	s := r.T
	for _, run := range r.Runs {
		s += run.T
	}
	return s
}

func (wb *workbook) sheet(name string) (sheet, error) {
	p, ok := wb.sheets[name]
	if !ok {
		return nil, fmt.Errorf("sheet %q not found", name)
	}
	var ws struct {
		Cells []struct {
			Ref    string   `xml:"r,attr"`
			Type   string   `xml:"t,attr"`
			Value  string   `xml:"v"`
			Inline richText `xml:"is"`
		} `xml:"sheetData>row>c"`
	}
	if err := wb.decode(p, &ws); err != nil {
		return nil, err
	}
	out := sheet{}
	for _, c := range ws.Cells {
		if c.Ref == "" {
			return nil, fmt.Errorf("sheet %q: cell without a reference", name)
		}
		switch c.Type {
		case "s":
			i, err := strconv.Atoi(c.Value)
			if err != nil || i < 0 || i >= len(wb.shared) {
				return nil, fmt.Errorf("%s!%s: bad shared string index %q", name, c.Ref, c.Value)
			}
			out[c.Ref] = cell{wb.shared[i]}
		case "inlineStr":
			out[c.Ref] = cell{c.Inline.String()}
		default:
			out[c.Ref] = cell{c.Value}
		}
	}
	return out, nil
}

type missingPartError string

func (e missingPartError) Error() string { return fmt.Sprintf("xlsx part %s is missing", string(e)) }

func isMissing(err error) bool { _, ok := err.(missingPartError); return ok }

func (wb *workbook) decode(name string, v any) error {
	f, err := wb.zip.Open(name)
	if err != nil {
		return missingPartError(name)
	}
	defer f.Close()
	if err := xml.NewDecoder(f).Decode(v); err != nil {
		return fmt.Errorf("parse %s: %w", name, err)
	}
	return nil
}
