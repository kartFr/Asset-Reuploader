--!strict
local Theme = {}

local DEFAULT_THEME_PARAMETERS: RawTheme = {
	Name = "",

    Font = "SourceSansPro",
	TextSize = 15,
    TextColor = { Enum.StudioStyleGuideColor.MainText.Name },
    DimmedTextColor = { Enum.StudioStyleGuideColor.DimmedText.Name },
	StrokeColor = Color3.new(),
    StrokeTransparency = 0.75,  -- varies based on theme: 1 for light, 0.75 for dark if not set

	BorderColor = { Enum.StudioStyleGuideColor.Border.Name },

	BackgroundColor = { Enum.StudioStyleGuideColor.Light.Name },
	BackgroundImage = "",

    ForegroundColor = { Enum.StudioStyleGuideColor.MainBackground.Name },
	ForegroundImage = "",

    UnselectedColor = { Enum.StudioStyleGuideColor.Button.Name },
    SelectedColor = { Enum.StudioStyleGuideColor.Button.Name, Enum.StudioStyleGuideModifier.Pressed.Name },

    ToggleBackgroundColor = { Enum.StudioStyleGuideColor.DialogMainButton.Name },
	ToggleImage = "rbxassetid://97413807029185",

	InputColor = { Enum.StudioStyleGuideColor.MainText.Name },
	InputImage = "rbxassetid://72207659330742",
    TipColor = { Enum.StudioStyleGuideColor.MainText.Name },
    TipImage = "rbxassetid://110240163333510",

	XColor = { Enum.StudioStyleGuideColor.MainText.Name },
	XImage = "rbxassetid://116654648035158",

	NotificationTransparency = 0,
    NotificationImage = "",
    NotificationBackgroundImage = "",
}

local ContentProvider = game:GetService("ContentProvider")
local HttpService = game:GetService("HttpService")

local currentTheme: Theme
local plugin: Plugin
local themes: RawTheme = { Studio = { Name = "Studio" } }

type RawTheme = { [string]: any }

export type Theme = {
	Name: string,
	Id: string,

	Font: Font,
	TextSize: number,
	TextColor: Color3,
	DimmedTextColor: Color3,
	StrokeColor: Color3,
	StrokeTransparency: number,

	BorderColor: Color3,

	BackgroundColor: Color3,
	BackgroundImage: string,

	ForegroundColor: Color3,
	ForegroundImage: string,

	UnselectedColor: Color3,
	SelectedColor: Color3,

	ToggleBackgroundColor: Color3,
	ToggleImage: string,

	InputColor: Color3,
	InputImage: string,
	TipColor: Color3,
	TipImage: string,

	XColor: Color3,
	XImage: string,

	NotificationTransparency: number,
	NotificationImage: string,
	NotificationBackgroundImage: string,
}

local function deepCopy<T>(original: T): T
	assert(type(original) == "table", "can only deep copy tables")

	local copy = {}
	for i, v in original :: any do
		copy[i] = (type(v) == "table" and deepCopy(v) or v)
	end

	return copy
end

local function getFont(rawThemeFont: { any } | string): Font
	if type(rawThemeFont) == "string" then return Font.fromName(rawThemeFont) end
	assert(#rawThemeFont > 3, "font options is too large")

	local name, weight, style = table.unpack(rawThemeFont)
		
	local weightEnum: Enum.FontWeight? = Enum.FontWeight:FromName(weight) -- ignore, selene and luau-lsp bug: https://github.com/Kampfkarren/selene/issues/619
	assert(weightEnum, "font has invalid weight")

	local styleEnum: Enum.FontStyle? = Enum.FontStyle:FromName(style)
	assert(styleEnum, "font has invalid weight")

	return Font.fromName(name, weightEnum, styleEnum)
end

local function normalizeRawTheme(rawTheme: RawTheme, themeId: string): Theme
	local studioTheme = settings().Studio.Theme :: StudioTheme
	local theme: { [string]: any } = {}

	local function normalizeValue(key: string, value: any): any
		if type(value) ~= "table" then return value end

		local valueType = type( (value :: { any })[1] )
		if valueType == "string" then
			if #value > 2 then error(`{key} could not create a StudioStyleGuide`) end

			local color, modifier = table.unpack(value :: any)

			local colorEnum: Enum.StudioStyleGuideColor? = Enum.StudioStyleGuideColor:FromName(color)
			if not colorEnum then error(`{key} could not create a StudioStyleGuide`) end

			local modifierEnum: Enum.StudioStyleGuideModifier?
			if modifier then
				modifierEnum = Enum.StudioStyleGuideModifier:FromName(modifier :: any)
				if not modifierEnum then error(`{key} has invalid StudioStyleGuideModifier`) end
			end

			return studioTheme:GetColor(colorEnum, modifierEnum)
		elseif valueType == "number" then
			if #value ~= 3 then error(`{key} needs 3 values to create a Color3`) end

			local R, G, B = table.unpack(value :: any)
			local success, result = pcall(Color3.fromRGB, R, G, B)
			if not success then error(`{key} could not create a Color3`) end
			return result
		end

		error(`{key} is an invalid type`)
	end

	for k, v in DEFAULT_THEME_PARAMETERS do
		if k == "Font" then
			theme[k] = rawTheme[k] and getFont(rawTheme[k]) or getFont(v)
			continue
		end

		if rawTheme[k] then
			theme[k] = normalizeValue(k, rawTheme[k])
			continue
		end

		if k == "StrokeTransparency" then
			theme[k] = studioTheme.Name == "Light" and 1 or 0.75
			continue
		end

		theme[k] = normalizeValue(k, v)
	end

	theme.Id = themeId
	return theme
end

local function preloadThemeAssets(theme: Theme)
	local assets = {
		theme.ForegroundImage, theme.BackgroundImage, theme.ToggleImage, theme.TipImage,
		theme.XImage, theme.NotificationImage, theme.NotificationBackgroundImage
	}
	task.spawn(ContentProvider.PreloadAsync, ContentProvider, assets)
end

local function save()
	assert(plugin, "theme needs to be initialized, call Theme.init() first")
	
	local themesToSave = {}
	for themeId, theme in themes do
		if themeId == "Studio" then continue end
		themesToSave[themeId] = theme
	end

	plugin:SetSetting("Themes", themesToSave)
end

function Theme.init(currentPlugin: Plugin)
	assert(not plugin, "theme is already initialized")
	plugin = currentPlugin

	local savedThemes = plugin:GetSetting("Themes") or {}
	for themeId, theme in savedThemes do
		themes[themeId] = theme
	end

	local savedThemeId = plugin:GetSetting("CurrentTheme")
	if themes[savedThemeId] then
		Theme.set(savedThemeId)
		return
	end

	Theme.set("Studio")
end

function Theme.getThemes(): { [string]: RawTheme }
	return themes
end

function Theme.get(): Theme
	return currentTheme
end

function Theme.set(themeId: string)
	local theme: RawTheme? = themes[themeId]
	assert(theme, "theme does not exist")

	currentTheme = normalizeRawTheme(theme, themeId)
	plugin:SetSetting("CurrentTheme", themeId)
	preloadThemeAssets(currentTheme)
end

function Theme.add(rawTheme: RawTheme): string
	assert(type(rawTheme["Name"]) == "string", "theme requires a name")

	local id = HttpService:GenerateGUID(false)
	while themes[id] do 
		id = HttpService:GenerateGUID(false) 
	end
	themes[id] = rawTheme

	local success, result = pcall(normalizeRawTheme, rawTheme, id)
	assert(success, result)

	save()
	return id
end

function Theme.updateFrame(frame: GuiObject)
	frame.BackgroundColor3 = currentTheme.ForegroundColor
	frame.BorderColor3 = currentTheme.BorderColor
end

function Theme.updateText(text: TextBox | TextLabel | TextButton, updateBackground: boolean?)
	text.FontFace = currentTheme.Font
	text.TextSize = currentTheme.TextSize
	text.TextColor3 = currentTheme.TextColor
	text.TextStrokeColor3 = currentTheme.StrokeColor
	text.TextStrokeTransparency = currentTheme.StrokeTransparency
	if text:IsA("TextBox") then text.PlaceholderColor3 = currentTheme.DimmedTextColor end

	if updateBackground then 
		Theme.updateFrame(text) 
	end
end

function Theme.remove(themeId: string)
	themes[themeId] = nil
	save()
end

return Theme
