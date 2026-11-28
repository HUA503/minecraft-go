# Java 26.3 parity tracker

Last checked: 2026-11-28 against the project's `vanillaCreativeTabOrder` and the local Java 26.3 client assets.

## Current batch

- Added the eight missing tree-leaf items (jungle, acacia, dark oak, mangrove, cherry, pale oak and both azalea leaf types) and ten sapling items (spruce, birch, jungle, acacia, dark oak, mangrove propagule, cherry, pale oak and both azalea types).
- The items appear in the Natural creative tab and use bundled 26.3 block textures. Saplings can be placed on supported soil; they use crossed-plane rendering and are replaceable. Leaves use cutout cube rendering, drop their corresponding sapling chance when broken normally, and drop themselves with shears. Unsupported saplings are removed when the soil is mined.
- Sapling growth, tree-specific generation, leaf distance/persistence/decay, exact loot probabilities and drops for sticks/apples are not implemented. These entries therefore do not mean tree ecology matches vanilla.
- Azalea bushes currently share the generic sapling crossed-plane renderer; their 26.3 multipart bush model and distinct top/side/plant geometry still need implementation.
- Added the Java 26.3 shelf mushroom in commit `b352209`: two size stages, side support/facing, bone-meal growth, stage-specific drops and the rabbit-stew recipe. Bouncing behavior and wandering-trader offers remain missing.
- The creative catalog contains 1,044 entries after these additions.

## Direct creative-tab mapping gaps

The table counts references in the official category lists that have no matching local registry entry. Items present in more than one tab count more than once. It measures catalog mapping only; it does not establish correct textures, behavior or stack limits.

| Category | Java 26.3 entries | Unmapped references |
|---|---:|---:|
| Building Blocks | 387 | 0 |
| Colored Blocks | 162 | 4 |
| Natural | 254 | 72 |
| Functional | 149 | 103 |
| Redstone | 67 | 37 |
| Tools & Utilities | 118 | 73 |
| Combat | 78 | 19 |
| Food & Drinks | 45 | 3 |
| Ingredients | 151 | 0 |
| Spawn Eggs | 89 | 79 |
| **Total references** | **1,500** | **390** |

## UI and runtime gaps

- Creative inventory panel and representative tab textures use Java 26.3 assets; tab positions and hit regions have source-level checks. A full Java-client screenshot comparison at matching resolution and UI scale has not been completed, so pixel parity is unverified.
- Search/catalog counts, mouse behavior, tooltips, scrolling, player preview, other inventory screens, fonts and UI scaling still need visual and interaction comparison against 26.3.
- Rendering, items and blocks still have the tab-mapping gaps above; registered items are not necessarily behavior-complete.
- Crash reports from actual gameplay have not all been reproduced and closed. Save recovery, renderer stability and long-session behavior remain open verification areas.
- Achievements/advancements, multiplayer synchronization, complete entity ecology, world generation and protocol interoperability are not at vanilla parity.
