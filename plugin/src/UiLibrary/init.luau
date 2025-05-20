--!strict
local UiLibrary = {}

local Tab = require("./UiLibrary/Tab")
local Theme = require("./UiLibrary/Theme")
local Tip = require("./UiLibrary/Tip")

local asset = script.Assets.MainFrame

type ThemeManager = typeof(Theme)

local function selectTab(self: Ui, tabIndex: number)
	if self._currentTabIndex ~= 0 then
		local tabToUnselect = self._tabs[self._currentTabIndex]
		tabToUnselect.button:ColorUnselected()
		tabToUnselect.instance.Visible = false
	end

	local tabToSelect = self._tabs[tabIndex]
	tabToSelect.button:ColorSelected()
	tabToSelect.instance.Visible = true

	self._currentTabIndex = tabIndex
end

local function CreateTab(self: Ui, name: string): Tab.Tab
	local tabIndex = #self._tabs + 1
	local tab = Tab.new(self.instance :: any, function()
		if self._currentTabIndex == tabIndex then return end
		selectTab(self, tabIndex)
	end, name)
	table.insert(self._tabs, tab)

	local topbar = self.instance.Topbar
	local padding = topbar.Frame.UIPadding
	local gridLayout = topbar.Frame.UIGridLayout
	local tabCount = #topbar.Frame:GetChildren() - 2
	gridLayout.CellSize = UDim2.new(1 / tabCount, tabCount > 1 and -2 or 0, 1, 0)

	local isEvenTabs = tabCount % 2
	padding.PaddingRight = UDim.new(0, (2 + isEvenTabs) + math.max(0, tabCount - 3 - isEvenTabs))

	if self._currentTabIndex == 0 then
		selectTab(self, tabIndex)
	end
	return tab
end

local function CreateTip(self: Ui, text: string): Tip.Tip
	return Tip.new(self.instance.Tip :: any, text)
end

local function Notify(self: Ui, title: string, description: string)
	local notification = self.instance.Notification

	local mainFrame = notification.Main.Frame.Frame
	mainFrame.Description.Text = description
	mainFrame.Title.Text = title

	notification.Visible = true
end

local function UpdateTheme(self: Ui)
	local theme = Theme.get()
	
	local uiFrame = self.instance
	uiFrame.BackgroundColor3 = theme.BackgroundColor
	uiFrame.Image = theme.BackgroundImage

	local main = uiFrame.Main
	main.BackgroundColor3 = theme.BorderColor
	main.Frame.Image = theme.ForegroundImage
	main.Frame.BackgroundColor3 = theme.ForegroundColor

	local topBar = uiFrame.Topbar
	topBar.BackgroundColor3 = theme.BorderColor
	topBar.Frame.BackgroundColor3 = theme.ForegroundColor

	local notification = uiFrame.Notification
	notification.Image = theme.NotificationBackgroundImage
	notification.Main.Transparency = theme.NotificationTransparency
	notification.Main.BackgroundColor3 = theme.BorderColor

	local notificationBorder = notification.Main.Frame
	notificationBorder.BackgroundColor3 = theme.BorderColor

	local notificationMainFrame = notificationBorder.Frame
	Theme.updateText(notificationMainFrame.Close, true)
	Theme.updateText(notificationMainFrame.Description, true)
	Theme.updateText(notificationMainFrame.Title, true)
	notificationMainFrame.Title.TextSize += 10
	notificationMainFrame.Image = theme.NotificationImage
	notificationMainFrame.BackgroundColor3 = theme.ForegroundColor
	notificationMainFrame.Transparency = theme.NotificationTransparency

	local tipbar = uiFrame.Tip
	tipbar.BackgroundColor3 = theme.BorderColor
	tipbar.Frame.BackgroundColor3 = theme.ForegroundColor
	Theme.updateText(tipbar.Frame.TipText)

	for _, tab in self._tabs do
		tab:UpdateTheme()
	end

	if self._currentTabIndex ~= 0 then
		self._tabs[self._currentTabIndex].button:ColorSelected() 
	end
end

local function Destroy(self: Ui)
	for _, tab in self._tabs do
		tab:Destroy()
	end

	self._studioThemeChanged:Disconnect()
	self._notificationClosed:Disconnect()

	self.instance:Destroy()

	table.clear(self :: any)
end

export type Ui = {
	_currentTabIndex: number,
	_notificationClosed: RBXScriptConnection,
	_studioThemeChanged: RBXScriptConnection,
	_tabs: { Tab.Tab },

	theme: ThemeManager,
	instance: typeof(asset),

	CreateTab: typeof(CreateTab),
	CreateTip: typeof(CreateTip),
	Notify: typeof(Notify),
	UpdateTheme: typeof(UpdateTheme),
	Destroy: typeof(Destroy),
}

function UiLibrary.new(parent: Instance, plugin: Plugin): Ui
	local instance = asset:Clone()
	local notificationFrame = instance.Notification

	Theme.init(plugin)

	local self: Ui; self = {
		_currentTabIndex = 0,
		_notificationClosed = notificationFrame.Main.Frame.Frame.Close.MouseButton1Down:Connect(function()
			notificationFrame.Visible = false
		end),
		_studioThemeChanged = settings().Studio.ThemeChanged:Connect(function()
			Theme.set(Theme.get().Id)
			self:UpdateTheme()
		end),
		_tabs = {},

		theme = Theme,
		instance = instance,

		CreateTab = CreateTab,
		CreateTip = CreateTip,
		Notify = Notify,
		UpdateTheme = UpdateTheme,
		Destroy = Destroy,
	}

	self:UpdateTheme()
	instance.Parent = parent

	return self
end

return UiLibrary
