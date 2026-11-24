package validation

import (
	"net"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// formatChecker returns the profile's assertion checker for a vendored
// format, or nil when the profile treats it as annotation-only.
func formatChecker(format string) func(any) error {
	trim := func(v any) (string, bool) {
		s, ok := v.(string)
		return s, ok
	}
	simple := func(name string, ok func(string) bool) func(any) error {
		return func(v any) error {
			s, isStr := trim(v)
			if !isStr {
				return nil // non-strings are not format-checked per spec
			}
			if !ok(s) {
				return formatError(name, s)
			}
			return nil
		}
	}
	switch format {
	case "date-time":
		return simple("date-time", isDateTime)
	case "date":
		return simple("date", func(s string) bool {
			_, err := time.Parse("2006-01-02", s)
			return err == nil
		})
	case "time":
		return simple("time", isTime)
	case "duration":
		return simple("duration", isDurationShape)
	case "uri":
		return simple("uri", func(s string) bool {
			u, err := url.Parse(s)
			return err == nil && u.IsAbs() && strings.Contains(s, ":")
		})
	case "uri-reference":
		return simple("uri-reference", isURIReference)
	case "uri-template":
		return simple("uri-template", isURITemplateFormat)
	case "uuid":
		return simple("uuid", isUUID)
	case "json-pointer":
		return simple("json-pointer", isJSONPointer)
	case "relative-json-pointer":
		return simple("relative-json-pointer", isRelativeJSONPointer)
	case "ipv4":
		return simple("ipv4", func(s string) bool {
			ip := net.ParseIP(s)
			return ip != nil && ip.To4() != nil && strings.Count(s, ".") == 3
		})
	case "ipv6":
		return simple("ipv6", func(s string) bool {
			ip := net.ParseIP(s)
			return ip != nil && strings.Contains(s, ":")
		})
	case "email":
		return simple("email", isEmail)
	case "hostname":
		return simple("hostname", isHostname)
	case "regex":
		return simple("regex", func(s string) bool {
			_, err := regexp.Compile(s)
			return err == nil
		})
	default:
		return nil
	}
}

var timePattern = regexp.MustCompile(
	`^\d{2}:\d{2}:\d{2}(\.\d+)?([Zz]|[+-]\d{2}:\d{2})$`,
)
var timeDigits = regexp.MustCompile(`^\d{2}$`)

func isTime(s string) bool {
	if !timePattern.MatchString(s) {
		return false
	}
	parts := strings.SplitN(s, ":", 3)
	hour, minute := parts[0], parts[1]
	if !timeDigits.MatchString(hour) || !timeDigits.MatchString(minute) {
		return false
	}
	if hour > "23" || minute > "59" {
		return false
	}
	// Second field may carry a leap second (60) only at minute 59 (or
	// 23:59:60-shaped hours near midnight per RFC 3339 §5.7);
	secField := parts[2]
	sec := strings.SplitN(strings.SplitN(secField, ".", 2)[0], "Z", 2)[0]
	sec = strings.SplitN(strings.SplitN(sec, "z", 2)[0], "+", 2)[0]
	sec = strings.SplitN(sec, "-", 2)[0]
	if !timeDigits.MatchString(sec) || sec > "60" {
		return false
	}
	if sec == "60" {
		// Offset-aware leap-second rule: :60 is valid only when the
		// UTC-adjusted time is exactly 23:59 (RFC 3339 §5.7).
		offsetMinutes := 0
		if at := strings.IndexAny(secField, "+-"); at >= 0 {
			sign := 1
			if secField[at] == '-' {
				sign = -1
			}
			off := secField[at+1:]
			if len(off) != 5 || off[2] != ':' {
				return false
			}
			offHour, errH := strconv.Atoi(off[:2])
			offMinute, errM := strconv.Atoi(off[3:])
			if errH != nil || errM != nil {
				return false
			}
			offsetMinutes = sign * (offHour*60 + offMinute)
		}
		hourNum, errH := strconv.Atoi(hour)
		minNum, errM := strconv.Atoi(minute)
		if errH != nil || errM != nil {
			return false
		}
		utcTotal := (hourNum*60 + minNum - offsetMinutes + 24*60*2) % (24 * 60)
		if utcTotal/60 != 23 || utcTotal%60 != 59 {
			return false
		}
	}
	// Offset hour/minute sanity when present.
	if at := strings.IndexAny(secField, "+-"); at >= 0 {
		offset := secField[at+1:]
		if len(offset) != 5 || offset[2] != ':' {
			return false
		}
		offHour, offMinute := offset[:2], offset[3:]
		if !timeDigits.MatchString(offHour) || !timeDigits.MatchString(offMinute) {
			return false
		}
		if offHour > "23" || offMinute > "59" {
			return false
		}
	}
	return true
}

func isDurationShape(s string) bool {
	if !durationPattern.MatchString(s) {
		return false
	}
	body := strings.TrimPrefix(strings.TrimPrefix(s, "-"), "P")
	if body == "" {
		return false
	}
	if strings.Contains(body, "W") {
		// Weeks cannot combine with other units.
		return body == regexp.MustCompile(`^\d+W$`).FindString(body)
	}
	if at := strings.IndexByte(body, 'T'); at >= 0 {
		if at == len(body)-1 {
			return false
		}
	}
	return true
}

var uuidPattern = regexp.MustCompile(
	`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`,
)
var jsonPointerPattern = regexp.MustCompile(`^(/([^~/]|~[01])*)*$`)
var hostnameLabel = regexp.MustCompile(`^[A-Za-z0-9]([A-Za-z0-9-]{0,61}[A-Za-z0-9])?$`)
var durationPattern = regexp.MustCompile(
	`^-?P(\d+Y)?(\d+M)?(\d+W)?(\d+D)?(T(\d+H)?(\d+M)?(\d+(\.\d+)?S)?)?$`,
)

func isUUID(s string) bool        { return uuidPattern.MatchString(s) }
func isJSONPointer(s string) bool { return jsonPointerPattern.MatchString(s) }

func isHostname(s string) bool {
	if len(s) > 253 || s == "" {
		return false
	}
	labels := strings.Split(strings.TrimSuffix(s, "."), ".")
	for _, label := range labels {
		if !hostnameLabel.MatchString(label) {
			return false
		}
	}
	return true
}

// isDateTime enforces RFC 3339 date-time with the suite's edge cases:
// case-insensitive T/Z, leap seconds (:60 at minute 59), and offset
// sanity beyond what time.Parse accepts (it rejects :60 outright).
var dateTimePattern = regexp.MustCompile(
	`^\d{4}-\d{2}-\d{2}[Tt]\d{2}:\d{2}:\d{2}(\.\d+)?([Zz]|[+-]\d{2}:\d{2})$`,
)

func isDateTime(s string) bool {
	if !dateTimePattern.MatchString(s) {
		return false
	}
	datePart := s[:10]
	if _, err := time.Parse("2006-01-02", datePart); err != nil {
		return false
	}
	return isTime(s[11:])
}

// isEmail enforces RFC 5321-ish shapes the suite asserts: unquoted
// dot-atom local parts (no leading/trailing/embedded double dots), or a
// quoted string; domain is a hostname or an address literal in brackets.
var emailAddr = regexp.MustCompile(
	`^"[^"]*"@([^\[\]]+|\[[0-9a-fA-F.:]+\])$`,
)

func isEmail(s string) bool {
	var local, domain string
	if strings.HasPrefix(s, "\"") {
		// Quoted local part: the separator is the @ after the closing
		// quote, not any @ inside the quotes.
		close := strings.Index(s[1:], "\"")
		if close < 0 {
			return false
		}
		at := strings.Index(s[close+1:], "@")
		if at < 0 {
			return false
		}
		local, domain = s[:close+1+at], s[close+2+at:]
		if !emailAddr.MatchString(local + "@" + domain) {
			return false
		}
	} else {
		var found bool
		local, domain, found = strings.Cut(s, "@")
		if !found {
			return false
		}
		if local == "" || strings.HasPrefix(local, ".") || strings.HasSuffix(local, ".") || strings.Contains(local, "..") {
			return false
		}
		if !atextPattern.MatchString(local) {
			return false
		}
	}
	if strings.HasPrefix(domain, "[") {
		return isAddressLiteral(domain)
	}
	return isHostname(domain) && strings.Contains(domain, ".")
}

var atextPattern = regexp.MustCompile(`^[A-Za-z0-9!#$%&'*+/=?^_` + "`" + `{|}~.-]+$`)

func isAddressLiteral(domain string) bool {
	if !strings.HasSuffix(domain, "]") {
		return false
	}
	inner := strings.TrimSuffix(strings.TrimPrefix(domain, "["), "]")
	if strings.HasPrefix(inner, "IPv6:") {
		ip := net.ParseIP(strings.TrimPrefix(inner, "IPv6:"))
		return ip != nil && strings.Contains(strings.TrimPrefix(inner, "IPv6:"), ":")
	}
	ip := net.ParseIP(inner)
	return ip != nil && ip.To4() != nil
}

// isRelativeJSONPointer: digits then an optional json-pointer, with no
// leading zero followed by more digits.
func isRelativeJSONPointer(s string) bool {
	var n int
	for n < len(s) && s[n] >= '0' && s[n] <= '9' {
		n++
	}
	if n == 0 {
		return false
	}
	if n > 1 && s[0] == '0' {
		return false
	}
	rest := s[n:]
	if rest == "" || rest == "#" {
		return true
	}
	return isJSONPointer(rest)
}

// isURIReference enforces RFC 3986 URI-reference shape. Go's url.Parse
// accepts several malformed inputs (spaces, bare "%"), so check the
// character classes explicitly.
var uriRefPattern = regexp.MustCompile(
	`^[A-Za-z][A-Za-z0-9+.-]*:([^/?#]*)([/?#][^\s]*)?$|^//[^/?#]*([/?#][^\s]*)?$|^[^:\s/?#]+([/?#][^\s]*)?$|^([/?#][^\s]*)?$`,
)

func isURIReference(s string) bool {
	if strings.ContainsAny(s, " \t\r\n\\") {
		return false
	}
	if strings.Contains(s, "%") && badPercent(s) {
		return false
	}
	return uriRefPattern.MatchString(s)
}

func badPercent(s string) bool {
	for i := 0; i+2 < len(s); i++ {
		if s[i] == '%' {
			hex := s[i+1 : i+3]
			valid := len(hex) == 2 && isHex(hex[0]) && isHex(hex[1])
			if !valid {
				return true
			}
			i += 2
		}
	}
	return false
}

func isHex(c byte) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}

// isURITemplateFormat checks RFC 6570 template shape as a *format*:
// templates may be relative (no scheme required), unlike the
// framework's ResourceTemplate.uriTemplate location rule, which demands
// an absolute scheme.
var templateChars = regexp.MustCompile(`^[A-Za-z0-9._~:/?#\[\]@!$&'()*+,;=-]*$`)

func isURITemplateFormat(s string) bool {
	if s == "" {
		return false
	}
	if strings.Count(s, "{") != strings.Count(s, "}") {
		return false
	}
	for _, expansion := range templateVars.FindAllString(s, -1) {
		if !templateVar.MatchString(expansion) {
			return false
		}
	}
	// Every non-expansion segment must be URI characters.
	replaced := templateVars.ReplaceAllString(s, "")
	return templateChars.MatchString(replaced)
}

type formatFailure struct{ msg string }

func (e *formatFailure) Error() string { return e.msg }

func formatError(format, value string) error {
	return &formatFailure{
		msg: "validation: " + format + ": value does not match the " + format + " format: " + clipErr(value),
	}
}

func clipErr(value string) string {
	if len(value) > 60 {
		return value[:60] + "…"
	}
	return value
}
