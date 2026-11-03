# patientmatchingorg-3m4p1

The [patientmatching.org](https://patientmatching.org/) specification
**v3.4.1** matching rules as a
[go-opm-core](https://github.com/PortobelloAuth/go-opm-core) `*rules.RuleSet`.

The module is named for the specification version it implements: `3m4p1` is
major `3`, minor (`m`) `4`, patch (`p`) `1`. Each supported specification
version is its own module, so the module's own version is independent of the
specification's.

## Usage

```go
rs, err := patientmatchingorg3m4p1.New() // *rules.RuleSet
```

`New` returns the 29 Category 1 (Direct Match) rules of Table 2, specification
section III.C. Rule 29 was removed in v3.4.1; its ID is retired. The rule table
is tested against the specification's machine-readable Table 2
(`testdata/spec-v3-4-1-table-2-category-1.json`, from
[Table 2 Downloads](https://patientmatching.org/docs/table-2-downloads)).

## Demographic keys

Rules use FHIR R4 names where FHIR has one. A translation layer from OIDC ID
token claims and FHIR Bundles is a later step.

| Spec field | Key |
| --- | --- |
| First Name | `given` |
| Last Name | `family` |
| DOB | `birthDate` |
| Street Line | `line` |
| Phone Number | `phone` |
| Email Address | `email` |
| SSN Last 4 | `ssnLast4` |
| ITIN Last 4 | `itinLast4` |
| MBI | `mbi` |
| Legal ID (w/ issuing-authority namespace) | `legalId` |
| Member ID (w/ Payer namespace) | `memberId` |
| Subscriber ID (w/ Payer namespace) | `subscriberId` |
| Namespace bound unique identifiers | `identifier` |

Rule labels are the keys joined by `-`, with the specification's `*` kept on
fields that may be fuzzy, for example `given*-family-birthDate-ssnLast4` (rule
04). Labels name fields, not rule numbers, so a renumbering does not change
what goes on the wire, and rules that differ only in which field may be fuzzy
stay distinct.

## Normalization

Each demographic gets go-opm-core normalizations, then this module's own
placeholder list (§V.D, `placeholders.go`; supplied through `placeholder.New`), so a placeholder value is reduced to the empty string.
Names are checked for placeholders before normalization as well, because the
name placeholders contain whitespace and punctuation.

## Not yet implemented

- **Fuzzy matching (§V.E).** Fields marked `*` match exactly. Their
  `demographicalts` functions (Damerau-Levenshtein distance 1 for names of 5 or
  more characters, ±1 day for DOB) do not exist yet.
- **Normalizations missing from go-opm-core main.** `dob_iso` (DOB),
  `phone_e164` (phone) and `digits_only` (SSN and ITIN last 4) are in
  go-opm-core PR #5. They are listed in `Pending`; `TestPending` fails once they
  are registered, so they get wired in.
- **Category 2** (Household + Individual rules).
- **Identifier namespaces.** `legalId`, `memberId`, `subscriberId` and
  `identifier` are expected to carry their namespace in the value; how it is
  encoded belongs to the translation layer.
- **Nicknames, all known names, and the generational suffix check** (§V.B).
  The suffix is a demographic compared only when both records have it.
- **The DOB range check** (§V.D.6: before current year - 120 or after today + 2
  days is unavailable).

## License

This project is licensed under the Apache License 2.0 - see the
[LICENSE](LICENSE.md) file for details.
