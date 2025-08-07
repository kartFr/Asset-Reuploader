--!strict
local App = {}

local PLUGIN_ID = "rbxassetid://89096096219225"

local UiLibrary = require("./UiLibrary")

local currentVersion = script.Parent.Parent.Version.Value
local tabOrder = {
    "Animation",
    "Sound",
    "Mesh",
    "Replace",
    "Filter",
    "Settings"
}

local function getLatestVersion(): string
    local success, objects = pcall(game.GetObjects, game, PLUGIN_ID)
	if not success then return currentVersion end
	
	local latestPlugin = (objects :: any)[1]
	local latestVersion = latestPlugin.Version.Value
	latestPlugin:Destroy()

    return latestVersion
end

function App.init(ui: UiLibrary.Ui, plugin: Plugin)
    for _, tabName in tabOrder do
        local tab = script.Tabs:FindFirstChild(tabName)
        assert(tab, "tab not found")
        task.spawn(require(tab), ui, plugin)
    end

    if currentVersion ~= getLatestVersion() then
        ui:Notify("Notification", "Plugin is out of date.")
    end
end

return App