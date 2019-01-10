HISTFILE=~/.histfile
HISTSIZE=5000
SAVEHIST=10000

# Changing dirs
setopt autocd

# Completion
setopt always_to_end
setopt AUTO_MENU           # Show completion menu on a succesive tab press.
setopt AUTO_LIST           # Automatically list choices on ambiguous completion.
setopt AUTO_PARAM_SLASH    # If completed parameter is a directory, add a trailing slash.
setopt AUTO_PARAM_KEYS
setopt COMPLETE_ALIASES
setopt complete_in_word
unsetopt MENU_COMPLETE     # Do not autoselect the first completion entry.
setopt glob_complete

# Expansion and Globbing
setopt extendedglob
setopt nomatch

# History
setopt HIST_IGNORE_ALL_DUPS
setopt HIST_REDUCE_BLANKS
setopt hist_ignore_space
setopt inc_append_history
#setopt share_history
setopt extended_history

# Input/Output
unsetopt correct_all  
setopt correct
unsetopt FLOW_CONTROL      # Disable start/stop characters in shell editor.
setopt interactive_comments
unsetopt hashdirs
#setopt nohashcmds

# Prompting
setopt prompt_subst

# Job Control
setopt notify

autoload -Uz compinit
compinit
autoload -U promptinit
promptinit
autoload -U run-help
autoload run-help-git
alias help=run-help
autoload -U colors
colors
autoload -U url-quote-magic
zle -N self-insert url-quote-magic
autoload -U up-line-or-beginning-search
autoload -U down-line-or-beginning-search
zle -N up-line-or-beginning-search
zle -N down-line-or-beginning-search

cdpath=(~/Projects)
fpath=(/usr/local/share/zsh/site-functions $fpath)

