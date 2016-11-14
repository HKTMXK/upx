package main

import (
	"fmt"
	"github.com/codegangsta/cli"
	"os"
	"runtime"
	"sort"
)

var cmds = []string{
	"login", "logout", "cd", "pwd", "get", "put", "sync",
	"ls", "rm", "switch", "info", "mkdir", "services", "auth",
}

var version = "v0.1.4"

func main() {
	app := cli.NewApp()
	app.Name = "upx"
	app.Usage = "a tool for managing files in UPYUN"
	app.Author = "Hongbo.Mo"
	app.Email = "zjutpolym@gmail.com"
	app.Version = fmt.Sprintf("%s %s/%s %s", version, runtime.GOOS,
		runtime.GOARCH, runtime.Version())
	app.Commands = make([]cli.Command, 0)

	sort.Strings(cmds)
	for _, cmd := range cmds {
		cm, exist := CmdMap[cmd]
		if exist {
			if cm.Flags == nil {
				cm.Flags = make(map[string]CmdFlag)
			}
			for k, v := range GlobalFlags {
				cm.Flags[k] = v
			}
			Cmd := cli.Command{
				Name:  cmd,
				Usage: cm.Desc,
				Action: func(c *cli.Context) error {
					opts := make(map[string]interface{})
					for k, v := range cm.Flags {
						if c.IsSet(k) {
							switch v.typ {
							case "bool":
								opts[k] = c.Bool(k)
							case "string":
								opts[k] = c.String(k)
							case "int":
								opts[k] = c.Int(k)
							}
						}
					}

					needUser := true
					if cmd == "login" || cmd == "logout" ||
						cmd == "switch" || cmd == "services" || cmd == "auth" {
						needUser = false
					}
					initDriver(c.String("auth"), needUser)
					if needUser && driver == nil {
						fmt.Println("Log in first.")
						os.Exit(-1)
					}
					cm.Func(c.Args(), opts)
					return nil
				},
			}
			if cm.Alias != "" {
				Cmd.Aliases = []string{cm.Alias}
			}
			if cm.Flags != nil {
				Cmd.Flags = []cli.Flag{}
				for k, v := range cm.Flags {
					var flag cli.Flag
					switch v.typ {
					case "bool":
						flag = cli.BoolFlag{Name: k, Usage: v.usage}
					case "string":
						flag = cli.StringFlag{Name: k, Usage: v.usage}
					case "int":
						flag = cli.IntFlag{Name: k, Usage: v.usage}
					}
					Cmd.Flags = append(Cmd.Flags, flag)
				}
			}

			app.Commands = append(app.Commands, Cmd)
		}
	}

	app.Run(os.Args)
}
