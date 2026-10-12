package domain

import (
	_ "embed"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
)

const (
	MinPasswordRunes = 12
	MaxPasswordRunes = 64
	MaxPasswordBytes = 72 // bcrypt reads no further
)

//go:embed passwords/common-10k.txt
var commonPasswordList string

// commonPasswords holds the breach list in normalized form, so a password is
// refused when it is a listed one dressed up: P@ssw0rd123! normalizes to password.
var commonPasswords = sync.OnceValue(func() map[string]struct{} {
	set := map[string]struct{}{}
	for line := range strings.SplitSeq(commonPasswordList, "\n") {
		for _, n := range normalizedForms(strings.TrimSpace(line)) {
			set[n] = struct{}{}
		}
	}
	return set
})

// ValidateNewPassword enforces the password policy: 12-64 characters (and at
// most 72 bytes for bcrypt), a lowercase and an uppercase ASCII letter, a digit
// and a symbol, no run or sequence of 4, no username / admin / portfolio, and
// nothing from the common-password list after normalization.
func ValidateNewPassword(username, password string) error {
	password = norm.NFC.String(password)
	n := utf8.RuneCountInString(password)
	switch {
	case n < MinPasswordRunes:
		return invalid("newPassword", "must be at least 12 characters")
	case n > MaxPasswordRunes:
		return invalid("newPassword", "must be at most 64 characters")
	case len(password) > MaxPasswordBytes:
		return invalid("newPassword", "must be at most 72 bytes")
	}
	if missing := missingClasses(password); len(missing) > 0 {
		return invalid("newPassword", "must contain "+strings.Join(missing, ", "))
	}
	lower := strings.ToLower(password)
	switch {
	case hasRepeat(lower, 4):
		return invalid("newPassword", "must not repeat a character 4 times in a row")
	case hasSequence(lower, 4):
		return invalid("newPassword", "must not contain a sequence such as 1234, abcd or qwer")
	case containsPersonalWord(lower, username):
		return invalid("newPassword", "must not contain the username, admin or portfolio")
	case isCommonPassword(password):
		return invalid("newPassword", "is too common: it appears in lists of leaked passwords")
	}
	return nil
}

func missingClasses(p string) []string {
	var lower, upper, digit, symbol bool
	for _, r := range p {
		switch {
		case r >= 'a' && r <= 'z':
			lower = true
		case r >= 'A' && r <= 'Z':
			upper = true
		case r >= '0' && r <= '9':
			digit = true
		case !unicode.IsLetter(r) && !unicode.IsDigit(r):
			symbol = true
		}
	}
	var missing []string
	for _, c := range []struct {
		ok   bool
		name string
	}{
		{lower, "a lowercase letter (a-z)"},
		{upper, "an uppercase letter (A-Z)"},
		{digit, "a digit (0-9)"},
		{symbol, "a symbol or space"},
	} {
		if !c.ok {
			missing = append(missing, c.name)
		}
	}
	return missing
}

func hasRepeat(s string, n int) bool {
	run, prev := 0, rune(-1)
	for _, r := range s {
		if r == prev {
			run++
		} else {
			run, prev = 1, r
		}
		if run >= n {
			return true
		}
	}
	return false
}

var keyboardRows = []string{"qwertyuiop", "asdfghjkl", "zxcvbnm", "1234567890"}

// hasSequence finds n characters that step by one in the alphabet or digits, in
// either direction, or that run along a keyboard row.
func hasSequence(s string, n int) bool {
	rs := []rune(s)
	for i := 0; i+n <= len(rs); i++ {
		w := rs[i : i+n]
		if alnum(w) && (steps(w, 1) || steps(w, -1)) {
			return true
		}
	}
	for _, row := range keyboardRows {
		for _, line := range []string{row, reverse(row)} {
			for i := 0; i+n <= len(line); i++ {
				if strings.Contains(s, line[i:i+n]) {
					return true
				}
			}
		}
	}
	return false
}

func alnum(w []rune) bool {
	for _, r := range w {
		if !(r >= 'a' && r <= 'z') && !(r >= '0' && r <= '9') {
			return false
		}
	}
	return true
}

func steps(w []rune, d rune) bool {
	for i := 1; i < len(w); i++ {
		if w[i]-w[i-1] != d {
			return false
		}
	}
	return true
}

func reverse(s string) string {
	r := []rune(s)
	for i, j := 0, len(r)-1; i < j; i, j = i+1, j-1 {
		r[i], r[j] = r[j], r[i]
	}
	return string(r)
}

func containsPersonalWord(lower, username string) bool {
	words := []string{DefaultAdminUsername, "portfolio"}
	if u := strings.ToLower(strings.TrimSpace(username)); len(u) >= 3 {
		words = append(words, u)
	}
	forms := append(normalizedForms(lower), lower)
	for _, f := range forms {
		for _, w := range words {
			if strings.Contains(f, w) {
				return true
			}
		}
	}
	return false
}

func isCommonPassword(p string) bool {
	set := commonPasswords()
	for _, n := range normalizedForms(p) {
		if _, ok := set[n]; ok {
			return true
		}
	}
	return false
}

// normalizedForms lowercases, strips leading and trailing digits and symbols
// (password2026! -> password), then undoes leetspeak. 1 reads as both i and l,
// so there are two forms. Forms shorter than 4 letters are dropped: too little
// is left to compare.
func normalizedForms(p string) []string {
	core := strings.TrimFunc(strings.ToLower(p), func(r rune) bool { return !unicode.IsLetter(r) })
	var out []string
	for _, one := range []rune{'i', 'l'} {
		f := strings.Map(func(r rune) rune {
			switch r {
			case '@', '4':
				return 'a'
			case '0':
				return 'o'
			case '1':
				return one
			case '!':
				return 'i'
			case '3':
				return 'e'
			case '5', '$':
				return 's'
			case '7':
				return 't'
			}
			return r
		}, core)
		if utf8.RuneCountInString(f) >= 4 && (len(out) == 0 || out[0] != f) {
			out = append(out, f)
		}
	}
	return out
}
