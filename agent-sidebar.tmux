set-option -gq @agent-sidebar-width "42"
set-option -gq @agent-sidebar-side "right"
set-option -gq @agent-sidebar-key "A"
set-option -gq @agent-sidebar-root "/home/no68/data/src/github.com/liubog2008/tmux-agent/main"
set-option -gq @agent-sidebar-bin "/home/no68/data/src/github.com/liubog2008/tmux-agent/main/bin/sidebar"
set-option -gq @agent-sidebar-state-bin "/home/no68/data/src/github.com/liubog2008/tmux-agent/main/bin/tmux-agent-state"

bind-key A run-shell "/home/no68/data/src/github.com/liubog2008/tmux-agent/main/scripts/toggle.sh"
