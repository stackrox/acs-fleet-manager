package services

import (
	"fmt"
	"regexp"
	"strings"

	"k8s.io/apimachinery/pkg/util/validation"
)

const (
	// Maximum namespace name length is 63
	// Namespace name is built using the central request id (always generated with 27 length) and the owner (truncated with this var).
	// Set the truncate index to 35 to ensure that the namespace name does not go over the maximum limit.
	replacementForSpecialChar = "-"
	appendChar                = "a"
	dns1123LabelFmt           = "[a-z0-9]([-a-z0-9]*[a-z0-9])?"
	dns1123LabelErrMsg        = "a lowercase RFC 1123 label must consist of lower case alphanumeric characters or '-', and must start and end with an alphanumeric character"
)

var (
	dns1123LabelRegexp = regexp.MustCompile("^" + dns1123LabelFmt + "$")
	// All OpenShift route hosts must confirm to DNS1035. This will inverse the validation RE from k8s (https://github.com/kubernetes/apimachinery/blob/master/pkg/util/validation/validation.go#L219)
	dns1035ReplacementRE = regexp.MustCompile(`[^a-z]([^-a-z0-9]*[^a-z0-9])?`)
)

func truncateString(str string, num int) string {
	truncatedString := str
	if len(str) > num {
		truncatedString = str[0:num]
	}
	return truncatedString
}

// maskProceedingandTrailingDash replaces the first and final character of a string with a subdomain safe
// value if is a dash.
func maskProceedingandTrailingDash(name string) string {
	if strings.HasSuffix(name, "-") {
		name = name[:len(name)-1] + appendChar
	}

	if strings.HasPrefix(name, "-") {
		name = strings.Replace(name, "-", appendChar, 1)
	}
	return name
}

// replaceHostSpecialChar replaces invalid characters with random char in the namespace name
func replaceHostSpecialChar(name string) (string, error) {
	replacedName := dns1035ReplacementRE.ReplaceAllString(strings.ToLower(name), replacementForSpecialChar)

	replacedName = maskProceedingandTrailingDash(replacedName)

	// This should never fail based on above replacement of invalid characters.
	validationErrors := validation.IsDNS1035Label(replacedName)
	if len(validationErrors) > 0 {
		return replacedName, fmt.Errorf("host name %q is not valid: a DNS-1035 label must consist of lower case alphanumeric characters or '-', start with an alphabetic character, and end with an alphanumeric character, regex used for validation is '[a-z]([-a-z0-9]*[a-z0-9])?'", strings.Join(validationErrors[:], ","))
	}

	return replacedName, nil
}

// FormatNamespace adds the rhacs prefix to the namespace name and performs the necessary validation and formatting to comply with RFC1123 that namespace names must follow.
func FormatNamespace(text string) (string, error) {
	if !dns1123LabelRegexp.MatchString(text) {
		return "", fmt.Errorf("invalid namespace %s: %s", text, validation.RegexError(dns1123LabelErrMsg, dns1123LabelFmt, "my-name", "123-abc"))
	}
	truncatedNamespace := truncateString("rhacs-"+text, validation.LabelValueMaxLength)
	return strings.TrimRight(truncatedNamespace, "-"), nil
}
