//go:build darwin || linux || freebsd || openbsd || netbsd

package main

import (
	"strings"
	"testing"
)

func TestShellQuote(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"/usr/local/bin/weakline", `'/usr/local/bin/weakline'`},
		{"/tmp/a b/weakline", `'/tmp/a b/weakline'`},
		{"/tmp/it's/weakline", `'/tmp/it'\''s/weakline'`},
		{"/tmp/$(rm -rf ~)/x", `'/tmp/$(rm -rf ~)/x'`},
	}

	for _, tt := range tests {
		if got := shellQuote(tt.in); got != tt.want {
			t.Errorf("shellQuote(%q) = %s, want %s", tt.in, got, tt.want)
		}
	}
}

func TestZshInitScript(t *testing.T) {
	script := zshInitScript("/opt/we akline/weakline")

	if strings.Contains(script, "@BIN@") {
		t.Error("placeholder was not replaced")
	}

	wants := []string{
		`_weakline_bin='/opt/we akline/weakline'`,
		"setopt PROMPT_SUBST",
		"_weakline_status=$?",
		"--async $$",
		"precmd_functions=(_weakline_precmd",
		"TRAPUSR1()",
		"return 0",
		`PROMPT='$("$_weakline_bin" $_weakline_status)'`,
	}
	for _, want := range wants {
		if !strings.Contains(script, want) {
			t.Errorf("init script is missing %q", want)
		}
	}
}
