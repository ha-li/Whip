package version_test

import (
	"testing"

	"github.com/ha-li/Whip/internal/app/version"
)

func TestAboutVersion(t *testing.T) {
	t.Parallel()

	for _, in := range []string{"1.4.2", "0.1.2-alpha"} {
		t.Run(in, func(t *testing.T) {
			t.Parallel()
			if got := version.NewAbout(version.MustParse(in)).Version(); got != in {
				t.Errorf("Version() = %q, want %q", got, in)
			}
		})
	}
}
