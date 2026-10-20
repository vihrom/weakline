//go:build darwin || linux || freebsd || openbsd || netbsd

package main

import (
	"fmt"
	"os"
	"strings"
)

// zshInitTemplate injects the shell integration. "@BIN@" is replaced by the
// shell-quoted absolute path of this executable (no PATH lookup per prompt).
//
//   - PROMPT_SUBST lets the prompt string run the renderer.
//   - precmd runs first and saves the exit status of the last command, so the
//     status stays correct when the prompt is redrawn later by TRAPUSR1.
//     It also starts the background git worker once per prompt (not on every
//     redraw such as a window resize); "&!" detaches it from job control.
//   - TRAPUSR1 redraws the prompt when the worker reports new data. It always
//     returns 0: zsh treats a non-zero trap status as an interrupt.
const zshInitTemplate = `
setopt PROMPT_SUBST
autoload -Uz add-zsh-hook

_weakline_bin=@BIN@
_weakline_status=0

_weakline_precmd() {
	_weakline_status=$?
	"$_weakline_bin" --async $$ </dev/null >/dev/null 2>&1 &!
}
precmd_functions=(_weakline_precmd ${precmd_functions:#_weakline_precmd})

TRAPUSR1() {
	zle && zle reset-prompt
	return 0
}

PROMPT='$("$_weakline_bin" $_weakline_status)'
`

func shellQuote(text string) string {
	return "'" + strings.ReplaceAll(text, "'", `'\''`) + "'"
}

func zshInitScript(exe string) string {
	return strings.Replace(zshInitTemplate, "@BIN@", shellQuote(exe), 1)
}

func handleInit(args []string) bool {
	if len(args) >= 2 && args[0] == "init" && args[1] == "zsh" {
		exe, err := os.Executable()
		if err != nil || exe == "" {
			exe = "weakline"
		}
		fmt.Print(zshInitScript(exe))
		return true
	}
	return false
}
