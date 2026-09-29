package validate

import (
	"fmt"
	"regexp"
)

var profileIDRe = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{0,63}$`)

func ProfileID(id string) error {
	if id == "" {
		return fmt.Errorf("profile_id is required")
	}
	if !profileIDRe.MatchString(id) {
		return fmt.Errorf("profile_id must match %s", profileIDRe.String())
	}
	return nil
}
