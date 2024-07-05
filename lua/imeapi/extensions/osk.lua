fcitx = require('fcitx')

local MSG_IME_ENTER = 'ImeEnter'
local MSG_IME_LEAVE = 'ImeLeave'

-- Compat stuff

if unpack == nil then
	unpack = table.unpack
end

-- Logging

local logPath = '/tmp/fcitx-osk.log'
local logFile, err = io.open(logPath, 'a')
if not logFile then
	print('[ERR] opening log file: ' .. err)
else
	-- logFile:setvbuf("line")
end

local function log(...)
	if not logFile then
		print(...)
		return
	end

	local ok, err = pcall(logFile.write, logFile, os.date(), ...)
	if not ok then
		print('[ERR] writing to log file: ' .. err)
	end

	ok, err = pcall(logFile.write, logFile, '\n')
	if not ok then
		print('[ERR] writing to log file: ' .. err)
	end
end

local function dbg(...) log('[DEBUG] ', ...) end
local function logError(...) log('[ERR] ', ...) end

print('hi there')
_ = logFile and logFile:write("wtf\n")

-- Utils

---Hacky ass function to find our daemon script
---@return string?
local function getDaemonPath()
	---@type string[]
	local files = fcitx.standardPathLocate(
		fcitx.StandardPath.Data,
		'fcitx5/osk-proxy',
		''
	)

	for _, file in ipairs(files) do
		if file:match('/daemon$') then
			return file
		end
	end

	return nil
end

-- Hooks

---@type {daemonIn: file*?}
local program = { daemonIn = nil }

local function initHooks()
	return {
		enterHook = fcitx.watchEvent(fcitx.EventType.InputMethodActivated, "OnIMEEnter"),
		leaveHook = fcitx.watchEvent(fcitx.EventType.InputMethodDeactivated, "OnIMELeave"),
		close = function(self)
			if self.enterHook then fcitx.unwatchEvent(self.enterHook) end
			if self.leaveHook then fcitx.unwatchEvent(self.leaveHook) end
		end,
	}
end

function _G.OnIMEEnter()
	local ok, err = assert(program.daemonIn):write(MSG_IME_ENTER, "\n")
	if not ok then
		logError(err)
	end
end

function _G.OnIMELeave()
	local ok, err = assert(program.daemonIn):write(MSG_IME_LEAVE, "\n")
	if not ok then
		logError(err)
	end
end

-- Sockets

local function main()
	dbg('start')
	local daemonFile = getDaemonPath()
	dbg('daemonFile ', daemonFile)
	if not daemonFile then
		logError('could not find daemon binary')
		os.exit(1)
	end
	program.daemonIn, err = io.popen(daemonFile .. ' 2> ' .. logPath, 'w')
	if not program.daemonIn then
		logError(err)
	end

	local _ = initHooks()
end

main()
