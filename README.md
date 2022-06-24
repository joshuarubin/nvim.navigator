# nvim.navigator

Replacement of

- [vim-tmux-navigator](https://github.com/christoomey/vim-tmux-navigator)
- [better-vim-tmux-resizer](https://github.com/RyanMillerC/better-vim-tmux-resizer)
- [wezterm.nvim](https://github.com/aca/wezterm.nvim)

for [wezterm](https://github.com/wez/wezterm).

## prerequisite

`~/.bashrc` / `~/.zshrc`

```sh
# linux
alias nvim="nvim --listen ${XDG_RUNTIME_DIR}/nvim-wezterm-pane-${WEZTERM_PANE}"

# mac
alias nvim="nvim --listen ${TMPDIR}/nvim-wezterm-pane-${WEZTERM_PANE}"
```

No plugins are required in neovim, all interaction takes place over rpc.

`~/.config/nvim/init.lua`

```lua
vim.keymap.set({ "n", "v" }, "<c-h>", "<c-w>h")
vim.keymap.set({ "n", "v" }, "<c-j>", "<c-w>j")
vim.keymap.set({ "n", "v" }, "<c-k>", "<c-w>k")
vim.keymap.set({ "n", "v" }, "<c-l>", "<c-w>l")

vim.keymap.set("t", "<c-h>", "<c-\\><c-n><c-w>h")
vim.keymap.set("t", "<c-j>", "<c-\\><c-n><c-w>j")
vim.keymap.set("t", "<c-k>", "<c-\\><c-n><c-w>k")
vim.keymap.set("t", "<c-l>", "<c-\\><c-n><c-w>l")

vim.keymap.set("i", "<c-h>", "<esc><c-w>h")
vim.keymap.set("i", "<c-j>", "<esc><c-w>j")
vim.keymap.set("i", "<c-k>", "<esc><c-w>k")
vim.keymap.set("i", "<c-l>", "<esc><c-w>l")
```

## navigator

Currently wezterm doesn't provide an api to manipulate wezterm remotely.
So we instead run a commandline program from wezterm which uses neovim's api.

Checkout https://github.com/wez/wezterm/discussions/995 for details/updates.

---

install

```sh
go install github.com/joshuarubin/nvim.navigator@latest
```

wezterm config

```lua
local function basename(s)
	local ret = string.gsub(s, "(.*[/\\])(.*)", "%2")
	if ret == nil then
		return ""
	end
	return ret
end

local function tmpdir()
	local dir = os.getenv("XDG_RUNTIME_DIR")
	if dir then
		return dir
	end

	dir = os.getenv("TMPDIR")
	if dir then
		return dir
	end

	return "/tmp"
end

local function nvim_socket(pane)
	return tmpdir() .. "/nvim-wezterm-pane-" .. pane:pane_id()
end

local function nvim_cmd(pane, dir, action)
	return os.getenv("HOME")
		.. "/go/bin/nvim.navigator"
		.. " -addr "
		.. nvim_socket(pane)
		.. " -dir "
		.. dir
		.. " -action "
		.. action
end

local function nvim_navigator(win, pane, dir, action)
	local p = io.popen(nvim_cmd(pane, dir, action), "r")
	if not p then
		win:toast_notification("wezterm", "failed to start nvim.navigator")
		return
	end

	local _, _, code = p:close()

	if code ~= 0 and code ~= 1 then
		win:toast_notification("wezterm", "nvim.navigator failed with code: " .. code)
	end

	return code
end

local function process_name(pane)
	local name = pane:get_foreground_process_name()
	return basename(name)
end

local function move_around(win, pane, direction_wez, direction_nvim)
	local name = process_name(pane)

	if name == "nvim" then
		local code = nvim_navigator(win, pane, direction_nvim, "move")
		if code == 0 then
			win:perform_action(wezterm.action({ SendKey = { mods = "CTRL", key = direction_nvim } }), pane)
			return
		end

		-- code 1 means vim couldn't move in the requested direction, anything
		-- else is an error
		if code ~= 1 then
			return
		end
	end

	win:perform_action(wezterm.action({ ActivatePaneDirection = direction_wez }), pane)
end

local function resize(win, pane, direction_wez, direction_nvim)
	if process_name(pane) == "nvim" then
		local code = nvim_navigator(win, pane, direction_nvim, "resize")
		if code == 0 then
			win:perform_action(wezterm.action({ SendString = "\x01" .. direction_nvim }), pane)
			return
		end

		-- code 1 means vim couldn't move in the requested direction, anything
		-- else is an error
		if code ~= 1 then
			return
		end
	end

	win:perform_action(wezterm.action({ AdjustPaneSize = { direction_wez, 1 } }), pane)
end
```

wezterm mapping

```lua
-- move panes (nvim aware)
{
	mods = "CTRL",
	key = "h",
	action = wezterm.action_callback(function(win, pane)
		move_around(win, pane, "Left", "h")
	end),
},
{
	mods = "CTRL",
	key = "j",
	action = wezterm.action_callback(function(win, pane)
		move_around(win, pane, "Down", "j")
	end),
},
{
	mods = "CTRL",
	key = "k",
	action = wezterm.action_callback(function(win, pane)
		move_around(win, pane, "Up", "k")
	end),
},
{
	mods = "CTRL",
	key = "l",
	action = wezterm.action_callback(function(win, pane)
		move_around(win, pane, "Right", "l")
	end),
},

-- resize panes (nvim aware)
{
	mods = "CTRL|SHIFT",
	key = "H",
	action = wezterm.action_callback(function(win, pane)
		resize(win, pane, "Left", "H")
	end),
},
{
	mods = "CTRL|SHIFT",
	key = "J",
	action = wezterm.action_callback(function(win, pane)
		resize(win, pane, "Down", "J")
	end),
},
{
	mods = "CTRL|SHIFT",
	key = "K",
	action = wezterm.action_callback(function(win, pane)
		resize(win, pane, "Up", "K")
	end),
},
{
	mods = "CTRL|SHIFT",
	key = "L",
	action = wezterm.action_callback(function(win, pane)
		resize(win, pane, "Right", "L")
	end),
},
```
