package main

// A simple Gio program. See https://gioui.org for more information.

import (
	"context"
	"fmt"
	"os"
	"strings"

	"gioui.org/ui"
	"gioui.org/ui/app"
	"gioui.org/ui/key"
	"gioui.org/ui/layout"
	"gioui.org/ui/measure"
	"gioui.org/ui/text"
	"github.com/go-redis/redis"

	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/sfnt"
)

func main() {
	w := app.NewWindow(
		app.WithWidth(ui.Dp(480)),
		app.WithHeight(ui.Dp(600)),
		app.WithTitle("redesk"),
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	a := &App{
		w:          w,
		ctx:        ctx,
		client:     nil,
		connected:  false,
		connection: nil,
		faces:      nil,
		ops:        nil,
	}

	if err := a.init(); err != nil {
		fmt.Println(err)
		os.Exit(2)
	}

	go func() {
		if err := a.run(); err != nil {
			fmt.Println(err)
			os.Exit(2)
		}
	}()

	app.Main()
}

// App...
type App struct {
	w *app.Window

	ctx context.Context

	client     *redis.Client
	connected  bool
	connection *Connection

	faces *measure.Faces
	ops   *ui.Ops

	editor  *text.Editor
	console *text.Editor
}

// Connection ...
type Connection struct {
	Name     string `json:"name"`
	Addr     string `json:"addr"`
	Password string `json:"password"`
	TLS      bool   `json:"tls"`
	Cluster  bool   `json:"cluster"`
}

func (a *App) init() error {
	mono, err := sfnt.Parse(gomono.TTF)
	if err != nil {
		return err
	}

	a.ops = new(ui.Ops)

	a.faces = &measure.Faces{}

	a.console = &text.Editor{
		Face:       a.faces.For(mono, ui.Sp(14)),
		Alignment:  text.Start,
		SingleLine: false,
		Submit:     true,
		Hint:       "$",
	}

	a.editor = &text.Editor{
		Face:       a.faces.For(mono, ui.Sp(14)),
		Alignment:  text.Start,
		SingleLine: true,
		Submit:     true,
		Hint:       "redis_addr:redis_port",
	}

	a.editor.SetText("localhost:6379")

	return nil
}

func (a *App) run() error {
	var cfg ui.Config

	for {
		e, ok := <-a.w.Events()
		if !ok {
			panic("not ok")
		}

		switch e := e.(type) {
		case key.Event:
			switch e.Name {
			case key.NameEscape:
				fmt.Println("esc - what do?")
			case key.NameDeleteBackward:
				if a.connected {
					txt := a.console.Text()
					if txt == "" {
						a.disconnect()
					}
				}
			case key.NameReturn:
				if a.connected {
					go a.redisCmd(a.console.Text())
					break
				}
				go a.connect(a.editor.Text())
			case 76:
				if e.Modifiers == key.ModCommand {
					a.console.SetText("")
					a.w.Invalidate()
				}
			default:
				fmt.Println("key", e.Name)
			}
		case app.DestroyEvent:
			return e.Err
		case *app.CommandEvent:
			switch e.Type {
			case app.CommandBack:
				e.Cancel = true
			}
		case app.UpdateEvent:
			cfg = &e.Config
			cs := layout.RigidConstraints(e.Size)
			a.paint(cs, cfg)
		}
	}
}

func (a *App) paint(cs layout.Constraints, cfg ui.Config) {
	queue := a.w.Queue()

	a.faces.Reset(cfg)
	a.ops.Reset()

	if a.connected {
		a.console.Layout(cfg, queue, a.ops, cs)
	} else {
		a.editor.Layout(cfg, queue, a.ops, cs)
	}

	a.w.Update(a.ops)
}

func (a *App) redisCmd(cmd string) {
	split := strings.Split(cmd, " ")
	args := make([]interface{}, len(split))
	for idx := range split {
		args[idx] = split[idx]
	}
	c := a.client.Do(args...)
	res, err := c.Result()
	if err != nil {
		fmt.Println(err)
		return
	}
	switch v := res.(type) {
	case []interface{}:
		strs := make([]string, len(v))
		for i := range v {
			strs[i] = v[i].(string)
		}
		s := "[" + strings.Join(strs, ",") + "]"
		a.console.SetText(s)
		a.w.Invalidate()
	case string:
		a.console.SetText(v)
		a.w.Invalidate()
		fmt.Println("invalidate!")
	default:
		fmt.Printf("%v - %T\n", v, v)
	}
}

func (a *App) connect(addr string) {
	client := redis.NewClient(&redis.Options{
		Network: "tcp",
		Addr:    addr,
	})

	pong, err := client.Ping().Result()
	if err != nil {
		a.sendErr(err)
		return
	}

	if pong != "PONG" {
		a.sendErr(fmt.Errorf("did not receive good PING response. got = %s", pong))
		return
	}

	fmt.Println("got pong?", pong)

	a.connection = &Connection{
		Name:     "test",
		Addr:     addr,
		Password: "",
		TLS:      false,
		Cluster:  false,
	}

	a.connected = true
	a.client = client

	a.w.Invalidate()
}

func (a *App) disconnect() {
	a.client.Close()
	a.connected = false
	a.client = nil
	a.w.Invalidate()
}

func (a *App) sendErr(err error) {
	fmt.Println("todo: send errors down to the 'client'. ", err)
}
