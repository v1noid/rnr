package cmd

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"rnr/utils"
	"strings"
	"syscall"

	"github.com/spf13/cobra"

	fs "github.com/fsnotify/fsnotify"
)

var startCommand = &cobra.Command{
	Use:   "run",
	Short: "Run any command with watch mode",
	Long: `To run any command with watch mode, use:
rnr run <command> [args...]`,

	Run: run,
}

const root string = "./"

type Config struct {
	IgnorePath []string `json:"ignorePath"`
	Command    string   `json:"command"`
	Separator  string   `json:"separator"`
	LogFlags   bool     `json:"logFlags"`
}

type CLI struct {
	args    []string
	Exec    *exec.Cmd
	Watcher *fs.Watcher
	Config  Config
}

func (c *CLI) Watch() {
	watcher, err := fs.NewWatcher()
	if err != nil {
		panic("Error watching filed: " + err.Error())
	}
	watcher.Add(root)
	filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {

		if d.IsDir() {
			err = watcher.Add(root + path)
		}
		return err
	})
	c.Watcher = watcher
}

func (c *CLI) ExecCommand() {
	if len(c.args) == 0 {
		log.Fatalf("No command provided to execute")
	}
	execCmd := &exec.Cmd{}

	execCmd = exec.Command(c.args[0], c.args[1:]...)

	fmt.Println(c.Config.Separator)
	execCmd.Stderr = os.Stderr
	execCmd.Stdout = os.Stdout
	err := execCmd.Start()
	if err != nil {
		log.Fatalf("Error starting command: %s", err.Error())
	}

	execCmd.Start()
	c.Exec = execCmd

}

func (c *CLI) RestartCommand() {
	c.Exec.Process.Kill()
	c.ExecCommand()
}

func (c *CLI) ReadConfig() {
	data, error := os.ReadFile("./rnr.config.json")

	if error != nil {
		log.Fatalf("Error reading config file: %s", error.Error())
	}
	error = json.Unmarshal(data, &c.Config)
	c.args = strings.Split(c.Config.Command, " ")
	if error != nil {
		log.Fatalf("Error parsing config file: %s", error.Error())
	}
	if c.Config.LogFlags == true {
		log.SetFlags(1)
	}
}

func (c *CLI) WatchEvents() {
	for {
		select {
		case event := <-c.Watcher.Events:
			if event.Op == fs.Write || event.Op == fs.Create || event.Op == fs.Remove {

				shouldRestart := !utils.Some(c.Config.IgnorePath, func(v string) bool {
					return strings.HasPrefix(event.Name, v)
				}) && event.Name != "rnr.config.json"

				if shouldRestart {
					c.RestartCommand()
				}
				if event.Name == "rnr.config.json" {
					c.ReadConfig()
					fmt.Println("\nConfig reloaded")
					c.RestartCommand()
				}
			}
		case err := <-c.Watcher.Errors:
			log.Fatalf("Error watching files: %s", err.Error())

		}
	}
}

func run(command *cobra.Command, args []string) {
	cli := &CLI{args: args}
	cli.ReadConfig()
	cli.Watch()
	defer cli.Watcher.Close()

	cli.ExecCommand()

	go cli.WatchEvents()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan
	fmt.Println("Closing")

}

func init() {
	rootCmd.AddCommand(startCommand)
}
