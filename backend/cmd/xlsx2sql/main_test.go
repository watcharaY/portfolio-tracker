package main

import (
	"archive/zip"
	"bytes"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// buildXLSX writes a minimal workbook. String values go to sharedStrings.xml,
// numbers to <v>, and a formula value carries its cached result like Excel does.
type formula struct{ cached string }

func buildXLSX(t *testing.T, sheets map[string]map[string]any) *bytes.Reader {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	put := func(name, body string) {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	names := slices.Sorted(maps.Keys(sheets))

	var shared []string
	var book, rels strings.Builder
	for i, name := range names {
		id := i + 1
		fmt.Fprintf(&book, `<sheet name="%s" sheetId="%d" r:id="rId%d"/>`, xmlEscape(name), id, id)
		fmt.Fprintf(&rels, `<Relationship Id="rId%d" Type="worksheet" Target="worksheets/sheet%d.xml"/>`, id, id)
		var cells strings.Builder
		for ref, v := range sheets[name] {
			switch v := v.(type) {
			case string:
				shared = append(shared, v)
				fmt.Fprintf(&cells, `<c r="%s" t="s"><v>%d</v></c>`, ref, len(shared)-1)
			case float64:
				fmt.Fprintf(&cells, `<c r="%s"><v>%s</v></c>`, ref, strconv.FormatFloat(v, 'g', -1, 64))
			case formula:
				fmt.Fprintf(&cells, `<c r="%s"><f>SUM(X1)</f><v>%s</v></c>`, ref, v.cached)
			default:
				t.Fatalf("unsupported cell value %T", v)
			}
		}
		put(fmt.Sprintf("xl/worksheets/sheet%d.xml", id),
			`<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData><row>`+cells.String()+`</row></sheetData></worksheet>`)
	}
	put("xl/workbook.xml", `<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><sheets>`+book.String()+`</sheets></workbook>`)
	put("xl/_rels/workbook.xml.rels", `<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">`+rels.String()+`</Relationships>`)
	var sst strings.Builder
	for _, s := range shared {
		fmt.Fprintf(&sst, "<si><t>%s</t></si>", xmlEscape(s))
	}
	put("xl/sharedStrings.xml", `<sst xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">`+sst.String()+`</sst>`)
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return bytes.NewReader(buf.Bytes())
}

func xmlEscape(s string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;").Replace(s)
}

func fixture() map[string]map[string]any {
	return map[string]map[string]any{
		"Brokerage": {
			"A5": "Broker", "B5": "Current Value",
			"A6": "Broker A", "B6": 1234.567,
			"A7": "Broker B's", "B7": 500.0,
			"B8": 999.0, // no name: skipped
			"I6": 45658.0, "J6": "Broker A", "K6": "Deposit", "L6": 1000.0, "M6": "first",
			"J7": "Broker A", "K7": "Withdraw", "L7": 250.5,
			"I8": 61.0, "J8": "Broker B's", "K8": "Deposit", "L8": 10.0,
			"I9": 59.0, "J9": "Broker B's", "K9": "Deposit", "L9": 20.0,
		},
		"Funds": {
			"A6": "Fund A", "B6": "Equity", "C6": 100.0, "D6": 110.0, "E6": formula{"10"}, "G6": "note",
		},
		"PVD": {
			"A6": 2025.0, "B6": 1000.0, "C6": 1000.0, "D6": 50.0, "E6": 50.0, "I6": formula{"2100"}, "J6": "Mixed",
		},
		"Savings": {
			"A6": "Bank A", "B6": 10000.0, "C6": 0.025, "E6": "",
			"A7": "Bank B", "B7": 200.0, "C7": 0.0125,
		},
		"Insurance": {
			"A6": "Insurer A", "D6": "life",
			"F6": "Insurer A", "G6": formula{"1"}, "I6": 3000.0,
			"F7": "Insurer A", "H7": 2025.0, "I7": 3000.0, "J7": "renewal",
		},
		"Salary & Tax 2026": {
			"A4": "Month", "M4": "Item",
			"A5": "Jan", "B5": 50000.0, "D5": 1234.5, "E5": 2500.0, "F5": 750.0,
			"A6": "Feb", "B6": formula{"50000"}, "C6": 10000.0,
			"M5": "Personal allowance", "N5": 60000.0,
			"N6": 1.0, // no item: skipped
		},
	}
}

func TestConvertRendersEveryTable(t *testing.T) {
	r := buildXLSX(t, fixture())
	got, _, err := convert(r, r.Size())
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"INSERT INTO brokers (name, current_value) VALUES\n  ('Broker A', 123457),\n  ('Broker B''s', 50000);\n",
		"INSERT INTO broker_transactions (txn_date, broker_id, type, amount, note) VALUES\n" +
			"  ('2025-01-01', (SELECT id FROM brokers WHERE name = 'Broker A'), 'deposit', 100000, 'first'),\n" +
			"  (NULL, (SELECT id FROM brokers WHERE name = 'Broker A'), 'withdraw', 25050, ''),\n" +
			"  ('1900-03-01', (SELECT id FROM brokers WHERE name = 'Broker B''s'), 'deposit', 1000, ''),\n" +
			"  ('1900-02-28', (SELECT id FROM brokers WHERE name = 'Broker B''s'), 'deposit', 2000, '');\n",
		"INSERT INTO funds (name, category, cost, market_value, note) VALUES\n  ('Fund A', 'Equity', 10000, 11000, 'note');\n",
		"INSERT INTO pvd_years (year, self_contribution, company_contribution, self_profit, company_profit, fund_allocation) VALUES\n  (2025, 100000, 100000, 5000, 5000, 'Mixed');\n",
		"INSERT INTO savings_accounts (bank, balance, interest_bp, note) VALUES\n  ('Bank A', 1000000, 250, ''),\n  ('Bank B', 20000, 125, '');\n",
		"INSERT INTO insurers (company, note) VALUES\n  ('Insurer A', 'life');\n",
		"INSERT INTO insurance_payments (insurer_id, year, premium, note) VALUES\n" +
			"  ((SELECT id FROM insurers WHERE company = 'Insurer A'), NULL, 300000, ''),\n" +
			"  ((SELECT id FROM insurers WHERE company = 'Insurer A'), 2025, 300000, 'renewal');\n",
		"INSERT INTO salary_months (year, month, salary, bonus, withholding_tax, pvd, sso) VALUES\n" +
			"  (2026, 1, 5000000, 0, 123450, 250000, 75000),\n" +
			"  (2026, 2, 5000000, 1000000, 0, 0, 0);\n",
		"INSERT INTO tax_deductions (year, item, amount) VALUES\n  (2026, 'Personal allowance', 6000000);\n",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing:\n%s\n--- in output ---\n%s", want, got)
		}
	}
	if !strings.HasPrefix(got[strings.Index(got, "BEGIN;"):], "BEGIN;\n\nDO $$") || !strings.HasSuffix(got, "\nCOMMIT;\n") {
		t.Errorf("seed must be one transaction that starts with the guard:\n%s", got)
	}
}

func TestGuardCoversEveryPortfolioTableButNotUsers(t *testing.T) {
	r := buildXLSX(t, fixture())
	got, _, err := convert(r, r.Size())
	if err != nil {
		t.Fatal(err)
	}
	guard := got[strings.Index(got, "DO $$"):strings.Index(got, "END $$;")]
	for _, table := range []string{"brokers", "broker_transactions", "funds", "pvd_years", "savings_accounts", "insurers", "insurance_payments", "salary_months", "tax_deductions"} {
		if !strings.Contains(guard, "EXISTS (SELECT 1 FROM "+table+")") {
			t.Errorf("guard does not check %s", table)
		}
	}
	if strings.Contains(got, "users") || strings.Contains(got, "sessions") {
		t.Error("seed must not touch users or sessions")
	}
}

func TestConvertRejectsBadCells(t *testing.T) {
	for name, tc := range map[string]struct {
		sheet, ref string
		value      any
		want       string
	}{
		"unknown txn type":   {"Brokerage", "K7", "Transfer", "Brokerage!K7 (broker_transactions.type)"},
		"phantom 1900-02-29": {"Brokerage", "I7", 60.0, "Brokerage!I7 (broker_transactions.txn_date)"},
		"bad month":          {"Salary & Tax 2026", "A6", "Febr", "Salary & Tax 2026!A6 (salary_months.month)"},
		"text money":         {"Funds", "C6", "lots", "Funds!C6 (funds.cost)"},
	} {
		t.Run(name, func(t *testing.T) {
			f := fixture()
			f[tc.sheet][tc.ref] = tc.value
			r := buildXLSX(t, f)
			_, _, err := convert(r, r.Size())
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("got %v, want error naming %q", err, tc.want)
			}
		})
	}
}
