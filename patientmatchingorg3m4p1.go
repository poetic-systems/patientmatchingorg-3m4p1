// Package patientmatchingorg3m4p1 provides the patientmatching.org specification
// v3.4.1 matching rules as a go-opm-core RuleSet.
package patientmatchingorg3m4p1

import (
	"fmt"
	"strings"

	"github.com/PortobelloAuth/go-opm-core/pkg/normalization"
	"github.com/PortobelloAuth/go-opm-core/pkg/normalization/placeholder"
	"github.com/PortobelloAuth/go-opm-core/pkg/rules"
)

// Version is the patientmatching.org specification version this module implements.
const Version = "3.4.1"

// Demographic keys. These are FHIR (R4 Patient / Coverage) names where FHIR
// has one; the identifier keys have no single FHIR element and are named here.
const (
	Given        = "given"        // Patient.name.given
	Family       = "family"       // Patient.name.family
	BirthDate    = "birthDate"    // Patient.birthDate
	Line         = "line"         // Patient.address.line (Street Line)
	Phone        = "phone"        // Patient.telecom where system = phone
	Email        = "email"        // Patient.telecom where system = email
	SSNLast4     = "ssnLast4"     // last 4 of the SSN identifier
	ITINLast4    = "itinLast4"    // last 4 of the ITIN identifier
	MBI          = "mbi"          // CMS-issued Medicare Beneficiary Identifier only
	LegalID      = "legalId"      // Legal ID, with issuing-authority namespace
	MemberID     = "memberId"     // Member ID, with payer namespace
	SubscriberID = "subscriberId" // Coverage.subscriberId, with payer namespace
	Identifier   = "identifier"   // Patient.identifier, namespace (system) bound
)

// normalizations lists, in order, how each demographic is normalized. A
// "placeholder-" name is one of this module's placeholder lists; the rest are
// registered go-opm-core normalizations. Placeholders are removed after the
// other normalizations. Names are also checked before them, because the name
// placeholders ("Test Patient", "Baby Boy", "N/A") contain the whitespace and
// punctuation that us-reduced_v0 removes.
var normalizations = map[string][]string{
	Given:        {"placeholder-names", "us-reduced_v0", "placeholder-names"},
	Family:       {"placeholder-names", "us-reduced_v0", "placeholder-names"},
	BirthDate:    {"placeholder-dateofbirth"},
	Line:         {"address", "placeholder-address"},
	Phone:        {"placeholder-phonenumber"},
	Email:        {"whitespace", "lowercase", "placeholder-email"},
	SSNLast4:     {"placeholder-last4"},
	ITINLast4:    {"placeholder-last4"},
	MBI:          {"us-reduced_v0"},
	LegalID:      {"whitespace", "lowercase"},
	MemberID:     {"whitespace", "lowercase", "placeholder-subscriberid"},
	SubscriberID: {"whitespace", "lowercase", "placeholder-subscriberid"},
	Identifier:   {"whitespace", "lowercase"},
}

// Pending lists the normalizations each demographic needs that go-opm-core
// main does not provide yet (go-opm-core PR #5). They belong before the
// placeholder list; until they land, these demographics match exactly as given.
var Pending = map[string][]string{
	BirthDate: {"dob_iso"},
	Phone:     {"phone_e164"},
	SSNLast4:  {"digits_only"},
	ITINLast4: {"digits_only"},
}

// category1 is Table 2, Category 1 (Direct Match Rules), specification
// section III.C. A trailing "*" marks a field that may be fuzzy. Rule 29 was
// removed in v3.4.1 and its ID is retired.
var category1 = []struct {
	id     string
	fields string
}{
	{"01", "given* family* birthDate* line*"},
	{"02", "given family* birthDate* phone"},
	{"03", "given* family* birthDate* email"},
	{"04", "given* family birthDate ssnLast4"},
	{"05", "given family* birthDate ssnLast4"},
	{"06", "given* family birthDate itinLast4"},
	{"07", "given family* birthDate itinLast4"},
	{"08", "given birthDate mbi"},
	{"09", "given birthDate legalId"},
	{"10", "family* birthDate* legalId"},
	{"11", "given birthDate phone"},
	{"12", "given birthDate email"},
	{"13", "given phone ssnLast4"},
	{"14", "given phone itinLast4"},
	{"15", "given email ssnLast4"},
	{"16", "given email itinLast4"},
	{"17", "phone mbi"},
	{"18", "phone legalId"},
	{"19", "email mbi"},
	{"20", "email legalId"},
	{"21", "legalId mbi"},
	{"22", "identifier"},
	{"23", "given birthDate memberId"},
	{"24", "family* birthDate* memberId"},
	{"25", "phone memberId"},
	{"26", "email memberId"},
	{"27", "given* family birthDate subscriberId"},
	{"28", "given family* birthDate subscriberId"},
	{"30", "family* birthDate phone"},
}

// New returns the Category 1 rules of the patientmatching.org specification
// v3.4.1. Each Rule is labeled with its fields joined by "-", keeping the "*"
// fuzzy marker so that rules differing only in which field may be fuzzy (04
// and 05, for example) stay distinct. Labels name fields, not rule numbers.
func New() (*rules.RuleSet, error) {
	seqs := make(map[string]*normalization.NormalizationSequence, len(normalizations))
	for k, names := range normalizations {
		seq, err := newSequence(names)
		if err != nil {
			return nil, fmt.Errorf("normalizations for %s: %w", k, err)
		}
		seqs[k] = seq
	}

	rs := &rules.RuleSet{Rules: make([]rules.Rule, 0, len(category1))}
	for _, r := range category1 {
		names := strings.Fields(r.fields)
		components := make([]rules.RuleDemographic, 0, len(names))
		for _, n := range names {
			k := strings.TrimSuffix(n, "*")
			seq, ok := seqs[k]
			if !ok {
				return nil, fmt.Errorf("rule %s: unknown demographic %s", r.id, k)
			}
			// TODO: fuzzy Alternatives for "*" fields (Damerau-Levenshtein 1 for
			// names of 5+ characters, ±1 day for DOB) are not built yet.
			components = append(components, rules.RuleDemographic{
				Label:          k,
				Key:            k,
				Normalizations: seq,
			})
		}
		rs.Rules = append(rs.Rules, rules.Rule{
			Label:      strings.Join(names, "-"),
			Components: components,
		})
	}
	return rs, nil
}

func newSequence(names []string) (*normalization.NormalizationSequence, error) {
	seq := &normalization.NormalizationSequence{Seq: make([]normalization.Normalization, 0, len(names))}
	for _, n := range names {
		var norm normalization.Normalization
		var err error
		if list, ok := placeholderLists[n]; ok {
			norm = placeholder.New(n, list)
		} else {
			norm, err = normalization.New(n)
		}
		if err != nil {
			return nil, err
		}
		seq.Seq = append(seq.Seq, norm)
	}
	return seq, nil
}
