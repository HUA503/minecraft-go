# Java 26.3 parity tracker

Last checked: 2026-11-28 against the project's `vanillaCreativeTabOrder` and the local Java 26.3 client assets.

## Current batch

- Added the eight missing tree-leaf items (jungle, acacia, dark oak, mangrove, cherry, pale oak and both azalea leaf types) and ten sapling items (spruce, birch, jungle, acacia, dark oak, mangrove propagule, cherry, pale oak and both azalea types).
- The items appear in the Natural creative tab and use bundled 26.3 block textures. Saplings can be placed on supported soil; they use crossed-plane rendering and are replaceable. Leaves use cutout cube rendering, drop their corresponding sapling chance when broken normally, and drop themselves with shears. Unsupported saplings are removed when the soil is mined.
- Sapling growth, tree-specific generation, leaf distance/persistence/decay, exact loot probabilities and drops for sticks/apples are not implemented. These entries therefore do not mean tree ecology matches vanilla.
- Azalea bushes currently share the generic sapling crossed-plane renderer; their 26.3 multipart bush model and distinct top/side/plant geometry still need implementation.
- Added the Java 26.3 shelf mushroom in commit `b352209`: two size stages, side support/facing, bone-meal growth, stage-specific drops and the rabbit-stew recipe. Bouncing behavior and wandering-trader offers remain missing.
- The creative catalog contains 1,063 entries after adding the 16 colored concrete powders, dandelion, poppy and wither rose.
- Added dandelion and poppy to the Natural tab with the official block textures, item sprites, plant placement rules, instant break and self-drops. Each now crafts into its vanilla dye. Bone-meal world-generation behavior remains unimplemented.
- Added the wither rose with its official item/block texture, plant placement and self-drop. Player contact applies Wither and deals periodic damage; mob contact effects are not yet simulated.
- Added all 16 dyed concrete-powder items with Java 26.3 textures and dye recipes. They fall through replaceable blocks in the current simulation and harden beside water into the matching concrete color.
- The 16 powder colors now sort at the vanilla concrete-powder group position in Java's dye-color order instead of falling to the end of the tab.
- Concrete-powder falling currently advances one block per game tick and does not create Java falling-block entities or preserve their full landing/drop behavior. This is functional coverage, not exact parity.
- Powder block-tick state is rebuilt from block data when chunks load and is isolated per dimension, so save/reload or dimension switches do not silently stop or misapply pending falls. Added regression coverage for both paths.

## Direct creative-tab mapping gaps

The table counts references in the official category lists that have no matching local registry entry. Items present in more than one tab count more than once. It measures catalog mapping only; it does not establish correct textures, behavior or stack limits.

| Category | Java 26.3 entries | Unmapped references |
|---|---:|---:|
| Building Blocks | 387 | 0 |
| Colored Blocks | 162 | 4 |
| Natural | 254 | 69 |
| Functional | 149 | 103 |
| Redstone | 67 | 37 |
| Tools & Utilities | 118 | 73 |
| Combat | 78 | 19 |
| Food & Drinks | 45 | 3 |
| Ingredients | 151 | 0 |
| Spawn Eggs | 89 | 79 |
| **Total references** | **1,500** | **387** |

## UI and runtime gaps

- Creative inventory panel and representative tab textures use Java 26.3 assets; tab positions and hit regions have source-level checks. A full Java-client screenshot comparison at matching resolution and UI scale has not been completed, so pixel parity is unverified.
- Search/catalog counts, mouse behavior, tooltips, scrolling, player preview, other inventory screens, fonts and UI scaling still need visual and interaction comparison against 26.3.
- Rendering, items and blocks still have the tab-mapping gaps above; registered items are not necessarily behavior-complete.
- The local `%APPDATA%/minecraft-go/crash.log` ends at 2026-11-23 and records a 3-second client UI stall: `Engine.Tick` held the engine mutex while natural spawning called the public mutex-taking `SpawnMob`. Current tick code uses the lock-aware `spawnMobLocked`; its deadlock regression passed 100 consecutive runs. Region corruption/interrupted-commit recovery cases passed 20 consecutive runs. These checks verify the logged failure path and deterministic storage cases, not all gameplay crashes or long-session stability.
- Go's race detector could not run in this Windows environment: `CGO_ENABLED=0`, with no `gcc` available on `PATH`. Race-level verification remains open.
- Save recovery beyond tested region scenarios, renderer stability and long-session behavior remain open verification areas.
- Achievements/advancements, multiplayer synchronization, complete entity ecology, world generation and protocol interoperability are not at vanilla parity.
