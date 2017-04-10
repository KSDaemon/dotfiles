completion='$(brew --prefix)/share/zsh/site-functions/_brew'

if test -f $completion
then
   source $completion
fi

completion='$(brew --prefix)/share/zsh/site-functions/_brew_cask'

if test -f $completion
then
   source $completion
fi

completion='$(brew --prefix)/share/zsh/site-functions/_brew_services'

if test -f $completion
then
   source $completion
fi

