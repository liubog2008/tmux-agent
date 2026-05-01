set-option -gq @agent-sidebar-width "42"
set-option -gq @agent-sidebar-side "right"
set-option -gq @agent-sidebar-key "A"
set-option -gq @agent-sidebar-bin "tmux-agent"

bind-key A run-shell "#{@agent-sidebar-bin} toggle"
