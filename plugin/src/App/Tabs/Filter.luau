--!strict
local UiLibrary = require("../../UiLibrary")

local MarketplaceService = game:GetService("MarketplaceService")

return function(ui: UiLibrary.Ui, plugin: Plugin)
	local tab = ui:CreateTab("Filter")
        local placeList

    _G.getPlaceList = function(): { number } -- guys im so sorry im just so drained :sob: will be back at full on v2.0.0, forgive me for _G use... I wont forgive myself
        local placesListValues = placeList:Get()
        local list = table.create(#placesListValues)
        for _, value in placesListValues do
            table.insert(list, tonumber(value))
        end
        return list :: { number }
    end
	
	local split1 = tab:CreateSplit()
    split1:CreateToggle("Search string values", function(state)
        plugin:SetSetting("StrVals", state)
    end, plugin:GetSetting("StrVals"))
    split1:CreateToggle("Search scripts", function(state)
        plugin:SetSetting("Scripts", state)
    end, plugin:GetSetting("Scripts"))
    
    local split2 = tab:CreateSplit()
    split2:CreateToggle("Search int/number values", function(state)
        plugin:SetSetting("NumVals", state)
    end, plugin:GetSetting("NumVals"))

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
