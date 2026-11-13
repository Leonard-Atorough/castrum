# Your first game

Let's make a small tank game. You'll drive with the keyboard, aim with the mouse, and defend yourself against three approaching tanks. Hit all three to win; let one reach you and the round is over.

Along the way you'll create entities, give them components, and write systems that turn input and game state into movement. You don't need to know Castrum's architecture already, but you should be comfortable editing Go code and running commands in a terminal.

We'll build a smaller version of the repository's [finished tank example](../../../../examples/your-first-game/main.go). The example adds waves, a reload timer, and more feedback; start there if you'd rather play first. This tutorial keeps the first game small enough to build and understand in stages.

1. [Set up the game](setup.md): create a project and open a window.
2. [Drive and aim](player.md): put a tank on screen, move it, and follow it with the camera.
3. [Make a playable round](combat.md): fire at enemies, detect hits, and display the result.
4. [Add sound and continue](sound.md): add music and a firing sound, then find places to take the game further.

Each chapter ends with a working program. Keep the files from the previous chapter as you go, and run commands from the project directory.
