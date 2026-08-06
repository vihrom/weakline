#  weakline

**weakline** is a blazingly fast, lightweight, and modern Zsh prompt written in Go. 

Designed for developers who value minimal latency, clean aesthetics, and instant terminal feedback.

![Screenshot](screenshot.jpg)

## Features

*  **Zero Lag:** Asynchronous Git status rendering to keep your terminal ultra-responsive.
*  **Clean Path Highlighting:** Distinct, readable colors for parent directories and your active folder.
*  **Environment Aware:** Seamlessly displays active Python virtual environments.
*  **Single Binary:** No heavy dependencies, complex shell frameworks, or bloated configs.

## Installation

1. Clone and build the binary
   ```bash
   git clone https://github.com/vihrom/weakline
   cd weakline
   go build -o ~/.local/bin/weakline main.go
   ```

2. Add this line to the end of your ~/.zshrc:
   ```bash
   eval "$(weakline init zsh)"
   ```

## Why Weakline? (vs Pure & Powerlevel10k)

There are already fantastic prompt engines in the Zsh ecosystem, most notably Pure and Powerlevel10k. However, Weakline was built to find the sweet spot between ultimate minimalism and rich Git diagnostics.

### Pure

Pure is the gold standard of minimal prompts - clean, elegant, and unobtrusive. However, its minimalism comes at the cost of Git visibility:

* It lacks granular Git indicators (staged, unstaged, untracked, ahead/behind counters) out of the box.

* When working in complex repositories or fast-paced workflows, you often have to manually run git status to see what’s actually happening.

*Weakline keeps Pure's clean aesthetic and lightweight feel, but gives you full, high-density Git insight at a glance.*

### Powerlevel10k

Powerlevel10k is an incredible engineering marvel - insanely fast, feature-rich, and infinitely configurable. But for many workflows, it’s simply over-engineered:

* Feature Bloat: Hundreds of segments (battery, Kubernetes, AWS, RAM, system load) that most developers never use.

* Configuration Overhead: Thousands of lines in .p10k.zsh, making custom tweaks tedious and brittle.

* Heavy Footprint: A massive codebase for a utility whose primary job is just to render a couple of lines in a terminal.

*Weakline cuts out the noise. No bloated configurations, no unnecessary segments. Just a single, fast Go binary providing directory context, Python venv status, and rich Git info.*

### Summary
Pure is beautiful, but lacks detailed Git context.
Powerlevel10k is powerful, but overly complex and heavy.
Weakline gives you detailed Git status, zero-lag async performance, and absolute simplicity.