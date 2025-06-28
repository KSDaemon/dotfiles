-- Pull in the wezterm API
local wezterm = require 'wezterm'
local act = wezterm.action

-- NOTE: resurrect plugin is not working at all in my Mac OS setup, but
-- let's keep it for the [potential] future use...

-- Some usefull plugins
-- Automatic Seesion Save/Restore
local resurrect = wezterm.plugin.require("https://github.com/MLFlexer/resurrect.wezterm")
resurrect.state_manager.set_max_nlines(1000)
-- resurrect doesn't understand ~ and relative paths
resurrect.state_manager.change_state_save_dir("/Users/kostik/.config/wezterm/")
resurrect.state_manager.periodic_save({
	interval_seconds = 60,
	save_workspaces = true,
	save_windows = true,
	save_tabs = true,
})

wezterm.on("resurrect.error", function(err)
    wezterm.log_error("Resurrect error: " .. tostring(err))
	wezterm.gui.gui_windows()[1]:toast_notification("resurrect", err, nil, 3000)
end)

wezterm.on("resurrect.state_manager.save_state.finished", function(state)
  wezterm.log_info("Resurrect save triggered")
  local json = wezterm.json_encode(state)
  wezterm.log_info("Saved state: " .. json)
end)

wezterm.on("trigger-resurrect-save", function(window, pane)
  local state = resurrect.workspace_state.get_workspace_state()
  wezterm.log_info("Manual save triggered")
  resurrect.state_manager.save_state(state)
  resurrect.window_state.save_window_action()
  wezterm.gui.gui_windows()[1]:toast_notification("Resurrect", "Workspace saved!", nil, 2000)
end)

--wezterm.on("gui-startup", resurrect.state_manager.resurrect_on_gui_startup)
wezterm.on("gui-startup", function(cmd)
  wezterm.log_info("GUI startup! Resurrecting...")
  resurrect.state_manager.resurrect_on_gui_startup(cmd)
end)

wezterm.on('window-config-reloaded', function(window, pane)
  window:toast_notification('wezterm', 'configuration reloaded!', nil, 4000)
end)



-- This will hold the configuration.
local config = wezterm.config_builder()

config.initial_cols = 120
config.initial_rows = 28
config.scrollback_lines = 5000
config.window_decorations = 'INTEGRATED_BUTTONS|RESIZE'

config.pane_focus_follows_mouse = true

config.font = wezterm.font("JetBrains Mono")
config.font_size = 13
config.color_scheme = 'Breath Silverfox (Gogh)'
--config.color_scheme = 'Dark+'
--config.color_scheme = 'Darkside'
--config.color_scheme = 'Tomorrow Night Eighties (Gogh)'
--config.color_scheme = 'Smyck (Gogh)'
--config.color_scheme = 'Palenight (Gogh)'
--config.color_scheme = 'OceanicNext (base16)'
--config.color_scheme = 'Materia (base16)'
--config.color_scheme = 'Modus Vivendi Tinted (Gogh)'
config.warn_about_missing_glyphs = false

config.keys = {
    { key = 'd', mods = 'SUPER', action = act.SplitHorizontal{ domain =  'CurrentPaneDomain' } },
    { key = 'd', mods = 'SUPER|SHIFT', action = act.SplitVertical{ domain =  'CurrentPaneDomain' } },
    { key = 'RightArrow', mods = 'SUPER', action = act.ActivateTabRelative(1) },
    { key = 'LeftArrow', mods = 'SUPER', action = act.ActivateTabRelative(-1) },
    { key = ']', mods = 'SUPER', action = act.ActivatePaneDirection 'Right' },
    { key = '[', mods = 'SUPER', action = act.ActivatePaneDirection 'Left' },
    { key = "S", mods = "CTRL|SHIFT", action = act.EmitEvent("trigger-resurrect-save") }
}

config.inactive_pane_hsb = {
  saturation = 0.9,
  brightness = 0.8,
}

config.window_background_opacity = 0.96
config.default_workspace = "default"


-- Finally, return the configuration to wezterm:
return config

