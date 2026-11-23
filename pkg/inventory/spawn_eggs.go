package inventory

// SpawnEggDefinition links a creative spawn egg to a mob already implemented
// by this game. Runtime IDs are project-local and are not Mojang registry IDs.
type SpawnEggDefinition struct {
	RuntimeID   int32
	EntityName  string
	Texture     string
	DisplayName string
}

var spawnEggDefinitions = []SpawnEggDefinition{
	{11000, "zombie", "zombie_spawn_egg", "僵尸刷怪蛋"},
	{11001, "skeleton", "skeleton_spawn_egg", "骷髅刷怪蛋"},
	{11002, "creeper", "creeper_spawn_egg", "苦力怕刷怪蛋"},
	{11003, "cow", "cow_spawn_egg", "牛刷怪蛋"},
	{11004, "pig", "pig_spawn_egg", "猪刷怪蛋"},
	{11005, "sheep", "sheep_spawn_egg", "绵羊刷怪蛋"},
	{11006, "chicken", "chicken_spawn_egg", "鸡刷怪蛋"},
	{11007, "enderman", "enderman_spawn_egg", "末影人刷怪蛋"},
	{11008, "spider", "spider_spawn_egg", "蜘蛛刷怪蛋"},
}

var spawnEggByID = func() map[int32]SpawnEggDefinition {
	definitions := make(map[int32]SpawnEggDefinition, len(spawnEggDefinitions))
	for _, definition := range spawnEggDefinitions {
		definitions[definition.RuntimeID] = definition
	}
	return definitions
}()

// SpawnEggDefinitions returns the supported spawn eggs in stable display order.
func SpawnEggDefinitions() []SpawnEggDefinition {
	return append([]SpawnEggDefinition(nil), spawnEggDefinitions...)
}

// SpawnEggByID resolves an internal item runtime ID.
func SpawnEggByID(runtimeID int32) (SpawnEggDefinition, bool) {
	definition, ok := spawnEggByID[runtimeID]
	return definition, ok
}

// SpawnEggTextureName returns the official resource name for a supported egg.
func SpawnEggTextureName(runtimeID int32) string {
	if definition, ok := SpawnEggByID(runtimeID); ok {
		return definition.Texture
	}
	return ""
}
