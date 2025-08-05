--!strict
local getFilterOptions = require("../GetFilterOptions")
local reuploadIds = require("../ReuploadIds")
local UiLibrary = require("../../UiLibrary")

local MarketplaceService = game:GetService("MarketplaceService")
local Selection = game:GetService("Selection")

return function(ui: UiLibrary.Ui, plugin: Plugin)
    local tab = ui:CreateTab("Sound")
    local placeList

    local function getPlaceList(): { number }
        local placesListValues = placeList:Get()
        local list = table.create(#placesListValues)
        for _, value in placesListValues do
            table.insert(list, tonumber(value))
        end
        return list :: { number }
    end

    tab:CreateButton("Reupload", function()
        local filter = getFilterOptions(plugin, game:GetDescendants())
        table.insert(filter.WhitelistedInstances, "Sound")

		reuploadIds(plugin, ui, filter, "Sound", getPlaceList())
	end)
	
	tab:CreateButton("Reupload Selected", function()
        local filter = {
            WhitelistedInstances = { "Sound", "StringValue", "NumberValue", "IntValue", "LuaSourceContainer" },
            Instances = Selection:Get()
        }

        reuploadIds(plugin, ui, filter, "Sound", getPlaceList())
	end)

    placeList = tab:CreateList()

	local placeInput = placeList:AddInput("Place Id", function(input)
		local placeId = tonumber(input)
        local success, productInfo = pcall(MarketplaceService.GetProductInfo, MarketplaceService, placeId)

        if not success then
            ui:Notify("Notification", "Error getting product info.") 
            return 
        end

        local assetTypeId = (productInfo :: any).AssetTypeId
		if assetTypeId ~= 9 then
            ui:Notify("Notification", "Invalid place id.") 
            return 
        end

        local stringPlaceId = tostring(placeId)
		if placeList.elements[stringPlaceId] then
            ui:Notify("Notification", "Place id is in list.") 
            return 
        end

		local element = placeList:CreateElement(stringPlaceId)
            :AddDestroyButton()
        
        element.instance.TextLabel.Text = `({placeId}) {(productInfo :: any).Name}`
	end)
	placeInput:AddTip(ui:CreateTip(
		"Reuploading needs a place id. Public games under the creator are found automatically."
	))

    local textBox = placeInput.instance.TextBox
    textBox:GetPropertyChangedSignal("Text"):Connect(function() 
        textBox.Text = string.gsub(textBox.Text, "%D", "") 
    end)
end
