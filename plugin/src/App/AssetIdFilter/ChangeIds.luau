--!strict
local ApiDump = require("../ApiDump")
local retry = require("../Retry")
local WaitGroup = require("../WaitGroup")

local ScriptEditorService = game:GetService("ScriptEditorService")
local AssetService = game:GetService("AssetService")

local cachedClassProperties: { [string]: { string } } = {}

export type IdPair = {
    newId: number,
    oldId: number
}

local function setScriptIds(scriptInstance: LuaSourceContainer, idsToChange: { IdPair })
    local source = scriptInstance.Source
    if not source then
        warn(scriptInstance, "has nil source")
        return
    end

    for _, idPair in idsToChange do
        local oldId = idPair.oldId
        local newId = idPair.newId
        source = string.gsub(source, `%f[%d]{oldId}%f[%D]`, tostring(newId))
    end

    if #source > 200_000 then
        ScriptEditorService:UpdateSourceAsync(scriptInstance, function() return source end) -- we do a bulk update because this function very very slow :/
    else
        scriptInstance.Source = source
    end
end

local function setAnimationId(animation: Animation, oldId: number, newId: number)
    animation.AnimationId = string.gsub(animation.AnimationId, tostring(oldId), tostring(newId)) 
end

local function setSoundId(sound: Sound, oldId: number, newId: number)
    sound.SoundId = string.gsub(sound.SoundId, tostring(oldId), tostring(newId))  
end

local function setNumberValueId(numValue: NumberValue | IntValue, _: number, newId: number)
    numValue.Value = newId
end

local function setStringValueIds(strValue: StringValue, oldId: number, newId: number)
    strValue.Value = string.gsub(strValue.Value, tostring(oldId), tostring(newId)) 
end

local function setCharacterMesh(characterMesh: CharacterMesh, _: number, newId: number)
    characterMesh.MeshId = newId
end

local function transferAttributes(oldInstance: Instance, newInstance: Instance)
    for name, value in oldInstance:GetAttributes() do
        newInstance:SetAttribute(name, value)
    end
end

local function transferTags(oldInstance: Instance, newInstance: Instance)
    for _, tag in oldInstance:GetTags() do
        newInstance:AddTag(tag)
    end
end

local function transferChildren(oldInstance: Instance, newInstance: Instance)
    for _, child in oldInstance:GetChildren() do
        if child:IsA("TouchTransmitter") then continue end
        child.Parent = newInstance
    end
end

local function transferProperties(oldInstance: Instance, newInstance: Instance)
    local className = oldInstance.ClassName
    if className ~= newInstance.ClassName then error(`oldInstance({className}) class is not equal to newInstance({newInstance.ClassName})`) end
    assert(ApiDump.isCached(), "no api dump cached")

    local cachedProperties = cachedClassProperties[className]
    if not cachedProperties then
        cachedProperties = ApiDump.getProperties(className)
        cachedClassProperties[className] = cachedProperties
    end

    for _, property in cachedProperties do
        if property == "Parent" or property == "Sandboxed" then continue end -- TODO: remove sandboxed when the update comes.
        (newInstance :: any)[property] = (oldInstance :: any)[property]
    end
end

local function transferJoints(oldInstance: Instance, newInstance: Instance)
    for _, instance in game:GetDescendants() do -- VERY BAD IMPLEMENTATION, just trying to push it out, will implement a more performant method in 2.0.0 update, it requires BIG revamps for what I have in mind...
        if not instance:IsA("JointInstance") then continue end
        if instance.Part0 == oldInstance then
			instance.Part0 = newInstance
		else
			instance.Part1 = newInstance
		end
    end
end

local function setMeshPart(meshPart: MeshPart, oldId: number, newId: number)
    local contentRetrieved, content: Content = retry(3, Content.fromAssetId, newId) 
	if not contentRetrieved then
		warn(`failed to get content from {newId}, skipping {oldId}`)
        return
	end
    
    local newMeshPart = AssetService:CreateMeshPartAsync(content, {
        CollisionFidelity = meshPart.CollisionFidelity,
        RenderFidelity = meshPart.RenderFidelity,
        FluidFidelity = meshPart.FluidFidelity
    } :: any)

    local success, result = pcall(function()
        transferProperties(meshPart, newMeshPart)
        transferAttributes(meshPart, newMeshPart)
        transferTags(meshPart, newMeshPart)
        transferChildren(meshPart, newMeshPart)
        transferJoints(meshPart, newMeshPart)
    end)
    if not success then
        newMeshPart:Destroy()
        error(result)
    end

    newMeshPart.Parent = meshPart.Parent
    meshPart:Destroy()
end

local function setSpecialMesh(specialMesh: SpecialMesh, oldId: number, newId: number)
    specialMesh.MeshId = string.gsub(specialMesh.MeshId, tostring(oldId), tostring(newId))
end

local instanceIdSetters = {
    Animation = setAnimationId,
    Sound = setSoundId,

    NumberValue = setNumberValueId,
    IntValue = setNumberValueId,
    StringValue = setStringValueIds,

    CharacterMesh = setCharacterMesh,
    MeshPart = setMeshPart,
    SpecialMesh = setSpecialMesh,
}

return function(filteredIds: { [number]: { Instance } }, idsToChange: { IdPair })
    local waitGroup = WaitGroup.new()
    local scriptIdChanges = {}

    for _, idPair in idsToChange do
        local oldId = idPair.oldId
        local newId = idPair.newId

        local instanceArray = filteredIds[oldId]
        if not instanceArray then continue end

        for _, instance in instanceArray do
            if instance:IsA("LuaSourceContainer") then
                if not scriptIdChanges[instance] then scriptIdChanges[instance] = {} end
                table.insert(scriptIdChanges[instance], idPair)
                continue
            end

            local className = instance.ClassName
            local setInstanceId = instanceIdSetters[className]
            if not setInstanceId then error(`{className} is not supported`) end

            waitGroup:Add(function()
                local success, result = pcall(setInstanceId :: any, instance, oldId, newId)
                if not success then warn("Failed to change", instance, `{oldId} to {newId}:`, result) end
            end)
        end
    end

    for instance, idPairs in scriptIdChanges do
        waitGroup:Add(function()
            local success, result = pcall(setScriptIds, instance, idPairs)
            if not success then warn("Failed to bulk change", instance, result) end
        end)  
    end

    waitGroup:Wait()
end
