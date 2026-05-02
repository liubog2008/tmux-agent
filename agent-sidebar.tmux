set-option -gq @agent-sidebar-width "42"
set-option -gq @agent-sidebar-side "right"
set-option -gq @agent-sidebar-key "A"
set-option -gq @agent-sidebar-new-window-key "N"
set-option -gq @agent-sidebar-agent-window-name "agent"
set-option -gq @agent-sidebar-agent-session-name "__agent__"
set-option -gq @agent-sidebar-bin "tmux-agent"
set-option -gq focus-events on
set-option -gq @agent-sidebar-status-format "#(#{@agent-sidebar-bin} status-segment)"

bind-key A run-shell "#{@agent-sidebar-bin} toggle"
bind-key N run-shell "#{@agent-sidebar-bin} new-window"

set-hook -g pane-focus-out 'run-shell "#{@agent-sidebar-bin} close --pane-id #{hook_pane}"'
