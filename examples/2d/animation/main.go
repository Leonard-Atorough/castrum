// Command animation loads embedded samurai spritesheets and demonstrates
// desired-state animation: looping idle/run, one-shot attack/hurt that
// transition back to idle. Run with go run .; -author regenerates the
// spritesheet metadata without opening a window.
package main

import (
	"embed"
	"flag"
	"fmt"

	"github.com/Leonard-Atorough/castrum"
	"github.com/Leonard-Atorough/castrum/animation"
	"github.com/Leonard-Atorough/castrum/asset"
	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/debug"
	"github.com/Leonard-Atorough/castrum/ebitrun"
	"github.com/Leonard-Atorough/castrum/geom"
	"github.com/Leonard-Atorough/castrum/input"
	"github.com/Leonard-Atorough/castrum/signal"
)

//go:embed ATTACK.png ATTACK.json HURT.png HURT.json IDLE.png IDLE.json RUN.png RUN.json
var files embed.FS

const (
	FPS   = 10
	Scale = 2
)

// strips maps each spritesheet to its region prefix and frame count
// (96x96 cells in a horizontal strip).
var strips = []struct {
	sheet  string
	prefix string
	frames int
}{
	{"IDLE.json", "idle", 10},
	{"RUN.json", "run", 16},
	{"ATTACK.json", "attack", 7},
	{"HURT.json", "hurt", 4},
}

func main() {
	output := flag.String("author", "", "regenerate spritesheet metadata into this directory without opening a window")
	flag.Parse()
	if *output != "" {
		if err := authorSheets(*output); err != nil {
			panic(err)
		}
		fmt.Printf("Saved spritesheets to %s\n", *output)
		return
	}
	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {
	g, err := newGame()
	if err != nil {
		return err
	}
	ebitrun.New(ebitrun.WithDebugOverlay()).Run(g)
	return nil
}

// buildSets is a [castrum.WithResourceProvider] constructor. It runs after the engine
// has provided the asset loader, so the strips can be loaded from it.
func buildSets(w *core.World) (animation.Sets, error) {
	loader, err := w.Resource[*asset.Loader]()
	if err != nil {
		return animation.Sets{}, err
	}
	tracks := make(map[string]animation.Track, len(strips))
	for _, strip := range strips {
		tr, err := track(loader, strip)
		if err != nil {
			return animation.Sets{}, err
		}
		tracks[strip.prefix] = tr
	}
	body := func(name string) []animation.Track { return []animation.Track{tracks[name]} }
	return animation.NewSets(map[string]animation.Set{
		"samurai": {
			Name: "samurai",
			States: map[string]animation.State{
				"idle":   {Name: "idle", Loop: animation.Loop, Tracks: body("idle")},
				"run":    {Name: "run", Loop: animation.Loop, Tracks: body("run")},
				"attack": {Name: "attack", Next: "idle", Tracks: body("attack")},
				"hurt":   {Name: "hurt", Next: "idle", EnterBlend: 0.1, Tracks: body("hurt")},
			},
		},
	})
}

// track builds a track over strip's frames, validating every region against
// the decoded sheet.
func track(loader *asset.Loader, strip struct {
	sheet  string
	prefix string
	frames int
}) (animation.Track, error) {
	sheet, err := asset.FileSheet(asset.ID(strip.sheet)).Resolve(loader)
	if err != nil {
		return animation.Track{}, err
	}
	texture, err := loader.Load[asset.TextureData](string(sheet.Texture))
	if err != nil {
		return animation.Track{}, err
	}
	if err := sheet.ValidateBounds(texture.Image.Bounds()); err != nil {
		return animation.Track{}, err
	}
	frames := make([]string, 0, strip.frames)
	for i := 0; i < strip.frames; i++ {
		name := fmt.Sprintf("%s_%d", strip.prefix, i)
		if _, err := sheet.Region(name); err != nil {
			return animation.Track{}, err
		}
		frames = append(frames, name)
	}
	return animation.Track{Name: strip.prefix, Source: asset.ID(strip.sheet), Frames: frames, FPS: FPS}, nil
}

func newGame() (*castrum.Game, error) {
	g := castrum.New(
		castrum.WithTitle("castrum - animation"),
		castrum.WithFilesystem(files),
		castrum.WithBindings(input.Bindings{
			"pause":  {input.KeyInput{Key: input.KeySpace}.Binding()},
			"attack": {input.KeyInput{Key: input.Key1}.Binding()},
			"hurt":   {input.KeyInput{Key: input.Key2}.Binding()},
			"run":    {input.KeyPairInput{Negative: input.KeyArrowLeft, Positive: input.KeyArrowRight}.Binding()},
		}),
		castrum.WithResourceProvider(buildSets),
	)

	sets, err := g.World().Resource[animation.Sets]()
	if err != nil {
		return nil, err
	}

	animator, err := animation.NewAnimator("samurai", "idle")
	if err != nil {
		return nil, err
	}

	samurai, err := g.World().NewEntity(
		animator,
		core.Transform{Scale: geom.Vector2{X: Scale, Y: Scale}},
	)
	if err != nil {
		return nil, err
	}
	if err := g.AddSystem(core.PhaseFrame, "animation.controls", playbackSystem(samurai)); err != nil {
		return nil, err
	}
	panel := g.World().MustResource[*debug.Panel]()
	panel.Status("controls", "LEFT/RIGHT: run   SPACE: pause   1: attack   2: hurt   (idle is the default)")
	completed := g.World().MustResource[*signal.Hub]().NewReader[animation.Completed]()
	if err := g.AddSystem(core.PhaseFixed, "animation.readout", func(ctx *core.Context) error {
		value, ok := samurai.Component[animation.Animator](ctx.World)
		if !ok {
			return fmt.Errorf("samurai has no animator component")
		}
		layers, err := value.Frames(sets)
		if err != nil {
			return err
		}
		region := ""
		if len(layers) > 0 {
			region = layers[len(layers)-1].Region
		}
		panel.Status("playback", fmt.Sprintf("state %s   frame %s   paused: %t   finished: %t",
			value.Current(), region, value.Paused, value.Finished()))
		for ev, ok := completed.Next(); ok; ev, ok = completed.Next() {
			if ev.Tick == ctx.Tick {
				panel.Message(fmt.Sprintf("%s completed a pass", ev.State))
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return g, nil
}

func playbackSystem(samurai core.Entity) core.SystemFunc {
	return func(ctx *core.Context) error {
		actions, _ := ctx.World.Resource[input.Actions]()
		value, ok := samurai.Component[animation.Animator](ctx.World)
		if !ok {
			return fmt.Errorf("samurai has no animator component")
		}
		next := ""
		flip := value.Style.FlipH
		axis := actions.Axis("run")
		if actions.JustPressed("attack") {
			next = "attack"
		} else if actions.JustPressed("hurt") {
			next = "hurt"
		} else if axis != 0 && (value.Current() == "idle" || value.Current() == "run") {
			// Movement only interrupts idle/run, never a one-shot attack/hurt.
			next, flip = "run", axis < 0
		} else if value.Current() == "run" && axis == 0 {
			next = "idle" // run key released
		}

		pause := actions.JustPressed("pause")
		if !pause && next == "" {
			return nil
		}
		return samurai.Update(ctx.World, func(value *animation.Animator) {
			if pause {
				value.Paused = !value.Paused
			}
			if next != "" {
				value.State = next
			}
			value.Style.FlipH = flip
		})
	}
}
