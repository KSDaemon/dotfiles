#!/usr/bin/env bash

source "$DOTFILES_ROOT/zsh-git-prompt/zshrc.sh"
ZSH_THEME_GIT_PROMPT_PREFIX=" %{$fg_bold[red]%}{%{$reset_color%}"
ZSH_THEME_GIT_PROMPT_SUFFIX="%{$fg_bold[red]%}}%{$reset_color%}"
ZSH_THEME_GIT_PROMPT_SEPARATOR="%{$fg_bold[red]%}|%{$reset_color%}"
ZSH_THEME_GIT_PROMPT_BRANCH="%{$fg_bold[cyan]%}"
ZSH_THEME_GIT_PROMPT_STAGED="%{$fg[green]%}%{●%G%}"
ZSH_THEME_GIT_PROMPT_CHANGED="%{$fg_bold[blue]%}%{✚%G%}"
ZSH_THEME_GIT_PROMPT_CLEAN="%{$fg_bold[green]%}%{✔%G%}%{$reset_color%}"
ZSH_THEME_GIT_PROMPT_BEHIND="%{%{$fg[cyan]%}↓%{$reset_color%}%G%}"
ZSH_THEME_GIT_PROMPT_AHEAD="%{%{$fg[cyan]%}↑%{$reset_color%}%G%}"
ZSH_GIT_PROMPT_PYBIN="`which python3`"

PROMPT='%B%{%F{green}%}%n%{%F{red}%}@%b%{%F{green}%}%m%B%{%F{green}%}:%{%F{magenta}%}%~/$(git_super_status)%(?.%B%{%F{green}%}.%{%F{red}%}!)%(!.#.>)%{$reset_color%} '
RPROMPT='%B%{%F{green}%}[%{%F{cyan}%}%T%{%F{green}%}] %{%F{blue}%}cmd#%(?.%B%{%F{green}%}.%{%F{red}%})%h%{$reset_color%}'
