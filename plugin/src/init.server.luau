--!strict
local UiLibrary = require("./Plugin/UiLibrary")
local App = require("./Plugin/App")

local RunService = game:GetService("RunService")

if not RunService:IsEdit() then
	return
end

local currentVersion = script.Parent.Version.Value

local toolbar = plugin:CreateToolbar("Asset Reuploader")
local widget = plugin:CreateDockWidgetPluginGui(
	"Asset Reuploader",
	DockWidgetPluginGuiInfo.new(Enum.InitialDockState.Float, false, true, 500, 250, 500, 250)
)

local toggleButton = toolbar:CreateButton("Toggle Gui", "Toggle gui visibility", "rbxassetid://139518452938878")
toggleButton.Click:Connect(function() widget.Enabled = not widget.Enabled end)

widget:GetPropertyChangedSignal("Enabled"):Connect(function() toggleButton:SetActive(widget.Enabled) end)
widget.Title = `Asset Reuploader {currentVersion}`

if plugin:GetSetting("Port") == nil then
	plugin:SetSetting("Port", 38073)
end

local ui = UiLibrary.new(widget, plugin)
App.init(ui, plugin)

plugin.Unloading:Once(function()
	ui:Destroy()
end)
