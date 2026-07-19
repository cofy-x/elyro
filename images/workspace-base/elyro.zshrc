# Elyro development shell

if [[ -n "${NO_COLOR:-}" || "${TERM:-}" == dumb ]]; then
  PROMPT='elyro:%m %~ ❯ '
else
  autoload -Uz colors && colors
  PROMPT='%F{blue}elyro:%m%f %F{cyan}%~%f %(?.%F{green}.%F{red})❯%f '

  if command -v dircolors >/dev/null 2>&1; then
    eval "$(dircolors -b)"
  fi
  alias ls='ls --color=auto'

  [[ -f /usr/share/zsh-autosuggestions/zsh-autosuggestions.zsh ]] && source /usr/share/zsh-autosuggestions/zsh-autosuggestions.zsh
  [[ -f /usr/share/zsh-syntax-highlighting/zsh-syntax-highlighting.zsh ]] && source /usr/share/zsh-syntax-highlighting/zsh-syntax-highlighting.zsh
fi
