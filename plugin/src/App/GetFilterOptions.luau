--!strict
local AssetIdFilter = require("./AssetIdFilter")

local pluginSettings: { [string]: { string } } = {
    StrVals = { "StringValue" },
    Scripts = { "LuaSourceContainer" },
    NumVals = { "NumberValue", "IntValue" }
}

return function(plugin: Plugin, filteredInstances: { Instance }): AssetIdFilter.FilterOptions
    local instanceArray: { string } = {}
    for settingName, classNames in pluginSettings do
        if not plugin:GetSetting(settingName) then continue end

        for _, className in classNames do
            table.insert(instanceArray, className)
        end
    end

    return {
        WhitelistedInstances = instanceArray,
        Instances = filteredInstances
    }
end
