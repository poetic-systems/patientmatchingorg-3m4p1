package patientmatchingorg3m4p1

import (
	"strings"

	"github.com/PortobelloAuth/go-opm-core/pkg/normalization"
	"github.com/PortobelloAuth/go-opm-core/pkg/normalization/placeholder"
	"golang.org/x/text/runes"
)

// Placeholder lists for specification v3.4.1 (§V.D). They live here, not in
// go-opm-core, because placeholder removal changes from one version to the next.
var notAsciiDigit = runes.Predicate(func(r rune) bool {
	return r < '0' || r > '9'
})

var notAsciiAlphaNum = runes.Predicate(func(r rune) bool {
	return !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z'))
})

var names = []placeholder.PlaceholderCheck{
	&placeholder.StringCheck{Value: "Unknown"},
	&placeholder.StringCheck{Value: "Unk"},
	&placeholder.StringCheck{Value: "Test"},
	&placeholder.StringCheck{Value: "Test Patient"},
	&placeholder.AndCheck{
		Checks: []placeholder.PlaceholderCheck{
			&placeholder.WithContextCheck{
				Fn: func(cfn normalization.ContextFunc) bool {
					v, ok := cfn("family")
					return ok && len(v) > 0
				},
			},
			&placeholder.OrCheck{
				Checks: []placeholder.PlaceholderCheck{
					&placeholder.PrefixStringCheck{Value: "Baby Boy"},
					&placeholder.PrefixStringCheck{Value: "Baby Girl"},
				},
			},
		},
	},
	&placeholder.StringCheck{Value: "Twin A"},
	&placeholder.StringCheck{Value: "Twin B"},
	&placeholder.StringCheck{Value: "Newborn"},
	&placeholder.StringCheck{Value: "N/A"},
	&placeholder.StringCheck{Value: "None"},
	&placeholder.StringCheck{Value: "TBD"},
	&placeholder.AndCheck{
		Checks: []placeholder.PlaceholderCheck{
			&placeholder.WithContextCheck{
				Fn: func(cfn normalization.ContextFunc) bool {
					v, ok := cfn("family")
					v = strings.ToUpper(v)
					return ok && v == "DOE"
				},
			},
			// paired with generic "John"/"Jane"
			&placeholder.OrCheck{
				Checks: []placeholder.PlaceholderCheck{
					&placeholder.StringCheck{Value: "John"},
					&placeholder.StringCheck{Value: "Jane"},
				},
			},
		},
	},
	// single-character are explicitly noted as placeholders to be removed by v3.4.1
	&placeholder.LengthCheck{Len: 1},
	// repeated-character strings ("X" "ZZZ")
	&placeholder.RepeatingCharacterCheck{},
}

var dateofbirth = []placeholder.PlaceholderCheck{
	// NOTE: using string checks here until the rules determine the need for
	// handling various different date formats
	&placeholder.StringCheck{Value: "01/01/1900"},
	&placeholder.StringCheck{Value: "01/01/1901"},
	&placeholder.StringCheck{Value: "01/01/1800"},
	&placeholder.StringCheck{Value: "01/01/2000"},
	&placeholder.StringCheck{Value: "11/11/1111"},
	&placeholder.StringCheck{Value: "01/01/0001"},
	// a DOB equal to the record's registration/admission date (common default-fill error)
}

var last4 = []placeholder.PlaceholderCheck{
	&placeholder.NotCheck{Check: &placeholder.LengthCheck{Len: 4}},
	&placeholder.StringCheck{Value: "1234"},
	&placeholder.NotCheck{Check: &placeholder.DigitsCheck{}},
	&placeholder.RepeatingCharacterCheck{},
}

var phonenumber = []placeholder.PlaceholderCheck{
	&placeholder.TransformCheck{
		Fn: func(s string) (string, error) {
			// remove non-digits
			var b strings.Builder
			b.Grow(len(s))
			for _, r := range s {
				if !notAsciiDigit.Contains(r) {
					b.WriteRune(r)
				}
			}
			return b.String(), nil
		},
		Check: &placeholder.OrCheck{
			Checks: []placeholder.PlaceholderCheck{
				// All the same digit
				&placeholder.RepeatingCharacterCheck{},
				// 123-456-7890
				&placeholder.StringCheck{Value: "1234567890"},
				// any number with a 555 area code - it does not exist
				// &PrefixStringCheck{"555"},
				// any number with a 555 exchange
				&placeholder.TransformCheck{
					Fn: func(s string) (string, error) {
						if len(s) == 7 {
							return s[0:3], nil
						}
						if len(s) == 10 {
							return s[3:6], nil
						}
						if len(s) == 11 && s[0] == '1' {
							return s[4:7], nil
						}

						return "", nil
					},
					Check: &placeholder.StringCheck{Value: "555"},
				},
			},
		},
	},
}

var email = []placeholder.PlaceholderCheck{
	&placeholder.StringCheck{Value: "test@test.com"},
	&placeholder.StringCheck{Value: "unknown@unknown.com"},
	&placeholder.StringCheck{Value: "none@none.com"},
	&placeholder.SuffixStringCheck{Value: "@example.com"},
	// The noreply check removes common non-alphanumeric characters first,
	// matching variations with punctuation in the identifier portion of the address
	&placeholder.TransformCheck{
		Fn: func(s string) (string, error) {
			var b strings.Builder
			b.Grow(len(s))
			for _, r := range s {
				if r == '@' || !notAsciiAlphaNum.Contains(r) {
					b.WriteRune(r)
				}
			}
			return b.String(), nil
		},
		Check: &placeholder.PrefixStringCheck{Value: "noreply@"},
	},
}

var address = []placeholder.PlaceholderCheck{
	&placeholder.StringCheck{Value: "123 Main St"}, // (unaccompanied by a real unit/city match),
	&placeholder.StringCheck{Value: "PO Box 0"},
	&placeholder.StringCheck{Value: "Unknown"},
	&placeholder.StringCheck{Value: "Homeless"},
	&placeholder.StringCheck{Value: "No Fixed Address"},
	// ZIP 00000 or 99999
	// TODO: use normalization/address/projectusat to parse the address and
	// check the zip (consider if this is too inefficient)
}

var subscriberid = []placeholder.PlaceholderCheck{
	&placeholder.StringCheck{Value: "PENDING"},
	&placeholder.StringCheck{Value: "TBD"},
	&placeholder.StringCheck{Value: "NONE"},
	// All-zero or all-nine strings
	// FIXME?: this implementation is a little more broad than the rule text
	&placeholder.RepeatingCharacterCheck{},
	// a payer's documented default/test enrollment value
}

// placeholderLists maps the name used in normalizations to its list.
var placeholderLists = map[string][]placeholder.PlaceholderCheck{
	"placeholder-names":        names,
	"placeholder-dateofbirth":  dateofbirth,
	"placeholder-last4":        last4,
	"placeholder-phonenumber":  phonenumber,
	"placeholder-email":        email,
	"placeholder-address":      address,
	"placeholder-subscriberid": subscriberid,
}
