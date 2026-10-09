package main

import "fmt"

// zshInitScript injects configuration hooks into the active shell environment.
// PROMPT_SUBST enables dynamic function evaluation inside the prompt string.
// TRAPUSR1 intercepts signals from background tasks to immediately refresh the visual grid.
const zshInitScript = `
setopt PROMPT_SUBST

TRAPUSR1() {
	zle && zle reset-prompt
}

PROMPT='$(weakline $$ $?)'
`

// handleInit prints shell integration code if requested.
func handleInit(args []string) bool {
	if len(args) >= 2 && args[0] == "init" && args[1] == "zsh" {
		fmt.Print(zshInitScript)
		return true
	}
	return false
}
