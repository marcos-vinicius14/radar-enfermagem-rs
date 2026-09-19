package job

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

func NormalizeText(s string) string {
	s = strings.ToLower(s)

	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	result, _, _ := transform.String(t, s)

	fields := strings.Fields(result)
	return strings.Join(fields, " ")
}

func Fingerprint(company, title, city string) string {
	val := strings.Join([]string{
		NormalizeText(company),
		NormalizeText(title),
		NormalizeText(city),
	}, "|")

	hash := sha256.Sum256([]byte(val))
	return hex.EncodeToString(hash[:])
}
