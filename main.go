package main

import (
	"./fth"
	"fmt"
	"github.com/chzyer/readline"
)

func main() {
	rl, err := readline.NewEx(&readline.Config{
		Prompt:                 "> ",
		HistoryFile:            "./history",
		DisableAutoSaveHistory: true,
	})
	if err != nil {
		panic(err)

	}
	defer rl.Close()
	for {
		fmt.Println(rl)
		line, err := rl.Readline()
		if err != nil {
			return
		}
		fmt.Println(line)
		rl.SaveHistory(line)
	}
	options := fth.NewOptions("init", true, 32, 32, 4096)
	fm := fth.NewForthMachine(options)
	fm.Init()
	fm.Run()
}
