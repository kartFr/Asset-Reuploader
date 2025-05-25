--!strict
local Input = {}

local Theme = require("../Theme")
local Tip = require("../Tip")

local asset = script.Parent.Parent.Assets.Input

local function AddTip(self: Input, tip: Tip.Tip)
    local textBox = self.instance.TextBox
    textBox.Position = UDim2.new(0, 25, 0.5, 0)
    textBox.Size = UDim2.new(1, -45, 1, -2)

    local tipFrame = tip.instance
    tipFrame.Position = UDim2.new(0, 5, 0.5, 0)
    tipFrame.Parent = self.instance

    self._tip = tip
end

local function UpdateTheme(self: Input)
    local theme = Theme.get()

    local instance = self.instance
    instance.BackgroundColor3 = theme.UnselectedColor
    instance.BorderColor3 = theme.BorderColor

    local button = instance.ImageButton
    Theme.updateFrame(button)
    button.BackgroundColor3 = theme.UnselectedColor
    button.Image = theme.InputImage
    button.ImageColor3 = theme.InputColor

    Theme.updateText(instance.TextBox, true)

    if self._tip then
        self._tip:UpdateTheme()
    end
end

local function Destroy(self: Input)
    if self._tip then self._tip:Destroy() end
    self.instance:Destroy()

    table.clear(self :: any)
end

export type Input = {
    _tip: Tip.Tip,

    instance: typeof(asset),

    AddTip: typeof(AddTip),
    UpdateTheme: typeof(UpdateTheme),
    Destroy: typeof(Destroy),
}

function Input.new(parent: GuiObject, placeholderText: string, callback: (input: string) -> ()): Input
    local instance = asset:Clone()
    local inputBox = instance.TextBox
    inputBox.PlaceholderText = placeholderText

    local function inputEntered()
        local text = inputBox.Text
        if text == "" then return end
        
        inputBox.Text = ""
        callback(text)
    end

    inputBox.FocusLost:Connect(function(enterPressed)
        if not enterPressed then return end
        inputEntered()
    end)

    local button = instance.ImageButton
    button.MouseButton1Down:Connect(function()
		button.BackgroundColor3 = Theme.get().SelectedColor
		inputEntered()
	end)

    local function colorUnselected() button.BackgroundColor3 = Theme.get().UnselectedColor end
    button.MouseButton1Up:Connect(colorUnselected)
    button.MouseLeave:Connect(colorUnselected)

    local self = {
        instance = instance,

        AddTip = AddTip,
        UpdateTheme = UpdateTheme,
        Destroy = Destroy,
    } :: Input

    self:UpdateTheme()
    instance.Parent = parent

    return self
end

return Input