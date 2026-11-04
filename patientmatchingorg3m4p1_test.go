package patientmatchingorg3m4p1

import (
	"encoding/json"
	"maps"
	"os"
	"slices"
	"strings"
	"testing"

	goopmcore "github.com/PortobelloAuth/go-opm-core"
	"github.com/PortobelloAuth/go-opm-core/pkg/match"
	"github.com/PortobelloAuth/go-opm-core/pkg/normalization"
)

// specKeys maps the field names used in Table 2 to demographic keys.
var specKeys = map[string]string{
	"First Name":    Given,
	"Last Name":     Family,
	"DOB":           BirthDate,
	"Street Line":   Line,
	"Phone Number":  Phone,
	"Email Address": Email,
	"SSN Last 4":    SSNLast4,
	"ITIN Last 4":   ITINLast4,
	"MBI":           MBI,
	"Legal ID":      LegalID,
	"Legal ID (w/issuing-authority namespace)":                                               LegalID,
	"Member ID (w/ Payer namespace)":                                                         MemberID,
	"Subscriber ID (w/ Payer namespace)":                                                     SubscriberID,
	"Namespace bound unique identifiers (e.g. EMPI, FHIR Patient Identifier, CSP UUID, etc)": Identifier,
}

// TestCategory1MatchesSpec checks the rule table against the machine-readable
// Table 2 published by the specification (testdata, from table-2-downloads).
func TestCategory1MatchesSpec(t *testing.T) {
	b, err := os.ReadFile("testdata/spec-v3-4-1-table-2-category-1.json")
	if err != nil {
		t.Fatal(err)
	}
	var spec struct {
		Version string              `json:"specification_version"`
		Rows    []map[string]string `json:"rows"`
	}
	if err := json.Unmarshal(b, &spec); err != nil {
		t.Fatal(err)
	}
	if spec.Version != Version {
		t.Fatalf("spec table is version %s, module implements %s", spec.Version, Version)
	}
	if len(spec.Rows) != len(category1) {
		t.Fatalf("spec has %d Category 1 rules, table has %d", len(spec.Rows), len(category1))
	}
	for i, row := range spec.Rows {
		want := make([]string, 0)
		for _, f := range strings.Split(row["Approved field combination"], " + ") {
			name, fuzzy := strings.CutSuffix(strings.TrimSpace(f), "*")
			k, ok := specKeys[name]
			if !ok {
				t.Fatalf("rule %s: no key for spec field %q", row["ID"], name)
			}
			if fuzzy {
				k += "*"
			}
			want = append(want, k)
		}
		r := category1[i]
		if r.id != row["ID"] {
			t.Errorf("row %d: id %s, spec %s", i, r.id, row["ID"])
		}
		if got := strings.Fields(r.fields); !slices.Equal(got, want) {
			t.Errorf("rule %s: fields %v, spec %v", r.id, got, want)
		}
	}
}

func TestNew(t *testing.T) {
	rs, err := New()
	if err != nil {
		t.Fatal(err)
	}
	flat, err := rs.Flatten()
	if err != nil {
		t.Fatal(err)
	}
	if len(flat) != len(category1) {
		t.Fatalf("flattened %d rules, want %d", len(flat), len(category1))
	}
	seen := make(map[string]bool)
	for i, fr := range flat {
		names := strings.Fields(category1[i].fields)
		if want := strings.Join(names, "-"); fr.Label != want {
			t.Errorf("rule %s: label %q, want %q", category1[i].id, fr.Label, want)
		}
		if seen[fr.Label] {
			t.Errorf("rule %s: duplicate label %q", category1[i].id, fr.Label)
		}
		seen[fr.Label] = true
		if len(fr.Components) != len(names) {
			t.Fatalf("rule %s: %d components, want %d", category1[i].id, len(fr.Components), len(names))
		}
		for j, c := range fr.Components {
			if want := strings.TrimSuffix(names[j], "*"); c.Key != want {
				t.Errorf("rule %s: component %d key %q, want %q", category1[i].id, j, c.Key, want)
			}
			if c.Normalizations == nil || len(c.Normalizations.Seq) == 0 {
				t.Errorf("rule %s: %s has no normalizations", category1[i].id, c.Key)
			}
		}
	}
}

func TestNormalizations(t *testing.T) {
	rs, err := New()
	if err != nil {
		t.Fatal(err)
	}
	seqs := make(map[string]*normalization.NormalizationSequence)
	for _, r := range rs.Rules {
		for _, c := range r.Components {
			seqs[c.Key] = c.Normalizations
		}
	}
	tests := []struct {
		key, in, want string
	}{
		{Given, "José", "jose"},
		{Given, "Mary-Ann", "maryann"},
		{Family, "O'Brien ", "obrien"},
		{Given, "Unknown", ""},
		{Given, "X", ""},
		{Given, "Test Patient", ""},
		{Given, "N/A", ""},
		{Given, "Twin A", ""},
		{Family, " Unknown ", ""},
		{Family, "ZZZ", ""},
		{BirthDate, "1980-02-29", "1980-02-29"},
		{BirthDate, "03/04/2007", "2007-03-04"},
		{Phone, "(212) 736-5000", "+12127365000"},
		{Phone, "1-212-736-5000", "+12127365000"},
		{Phone, "202-555-0100", ""},
		{Email, " Jane.Roe@Mail.org", "jane.roe@mail.org"},
		{Email, "Test@Test.com", ""},
		{Email, "no-reply@clinic.org", ""},
		{SSNLast4, "1234", ""},
		{SSNLast4, "4821", "4821"},
		{SSNLast4, "-4821 ", "4821"},
		{ITINLast4, "0000", ""},
		{MBI, "1EG4-TE5-MK73", "1eg4te5mk73"},
		{SubscriberID, "PENDING", ""},
		{MemberID, "W123 456 789-01", "w123456789-01"},
	}
	for _, tc := range tests {
		got, err := seqs[tc.key].Apply(tc.in)
		if err != nil {
			t.Errorf("%s %q: %v", tc.key, tc.in, err)
			continue
		}
		if got != tc.want {
			t.Errorf("%s %q: got %q, want %q", tc.key, tc.in, got, tc.want)
		}
	}
}

// TestUnavailable checks values that fail normalization: Match treats the
// error as an unavailable field, so the rule cannot match on it.
func TestUnavailable(t *testing.T) {
	rs, err := New()
	if err != nil {
		t.Fatal(err)
	}
	seqs := make(map[string]*normalization.NormalizationSequence)
	for _, r := range rs.Rules {
		for _, c := range r.Components {
			seqs[c.Key] = c.Normalizations
		}
	}
	for _, tc := range []struct{ key, in string }{
		{BirthDate, "01/01/1900"},
		{BirthDate, "1900-01-01"},
		{BirthDate, "11/11/1111"},
		{BirthDate, "not a date"},
		{Phone, "555-555-5555"},
		{Phone, "12"},
	} {
		if got, err := seqs[tc.key].Apply(tc.in); err == nil {
			t.Errorf("%s %q: got %q, want an error", tc.key, tc.in, got)
		}
	}
}

// TestDOBPlaceholder checks that a placeholder date is removed after
// dateofbirth normalizes it to ISO 8601: it has no value, and no error.
func TestDOBPlaceholder(t *testing.T) {
	rs, err := New()
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range rs.Rules {
		for _, c := range r.Components {
			if c.Key != BirthDate {
				continue
			}
			for _, in := range []string{"01/01/2000", "20000101", "2000-01-01"} {
				if got, err := c.Normalizations.Apply(in); err != nil || got != "" {
					t.Errorf("%q: got %q, %v; want removed placeholder", in, got, err)
				}
			}
			return
		}
	}
}

func TestPlaintextMatch(t *testing.T) {
	rs, err := New()
	if err != nil {
		t.Fatal(err)
	}
	a := map[string]string{
		Given: "José", Family: "O'Brien", BirthDate: "1980-02-29",
		Phone: "(212) 736-5000", Email: "Jane.Roe@Mail.org", SSNLast4: "4821",
	}
	with := func(kv ...string) map[string]string {
		m := maps.Clone(a)
		for i := 0; i < len(kv); i += 2 {
			if kv[i+1] == "" {
				delete(m, kv[i])
			} else {
				m[kv[i]] = kv[i+1]
			}
		}
		return m
	}
	for _, tc := range []struct {
		name string
		b    map[string]string
		want bool
	}{
		{"same record", a, true},
		{"formatting differs", with(Given, "jose", Family, "OBrien", BirthDate, "02/29/1980", Phone, "1-212-736-5000", SSNLast4, "4821"), true},
		{"only given+DOB (no rule)", map[string]string{Given: "José", BirthDate: "1980-02-29"}, false},
		{"given+DOB+phone (rule 11)", map[string]string{Given: "jose", BirthDate: "1980-02-29", Phone: "212.736.5000"}, true},
		{"given+family+DOB+SSN, different DOB", with(BirthDate, "1980-03-01", Phone, "", Email, ""), false},
		{"placeholder phone never matches", map[string]string{Given: "José", BirthDate: "1980-02-29", Phone: "555-555-5555"}, false},
		{"placeholder on both sides", map[string]string{Given: "Unknown", Phone: "202-555-0100"}, false},
		{"different person", map[string]string{Given: "Ann", Family: "Lee", BirthDate: "1990-01-01", SSNLast4: "9999"}, false},
	} {
		got, err := goopmcore.PlaintextMatch(&match.Demographics{Raw: a}, &match.Demographics{Raw: tc.b}, rs)
		if err != nil {
			t.Errorf("%s: %v", tc.name, err)
			continue
		}
		if got != tc.want {
			t.Errorf("%s: matched %v, want %v", tc.name, got, tc.want)
		}
	}
}
