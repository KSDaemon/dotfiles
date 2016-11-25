#!/usr/bin/env sh

source "$DOTFILES_ROOT/antigen/antigen.zsh"
antigen use oh-my-zsh
antigen bundle zsh-users/zsh-syntax-highlighting
#antigen bundle brew-cask
#antigen bundle brew
antigen bundle npm
antigen bundle grunt
antigen bundle symfony2
antigen bundle zsh-users/zsh-completions src
antigen apply
