/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package main

//import "github.com/SvenDH/go-card-engine/cmd"

import (
	"github.com/SvenDH/go-card-engine/godot"
	"graphics.gd/startup"

	"graphics.gd/classdb/SceneTree"
)

func main() {
	//cmd.Execute()

	startup.LoadingScene() // setup the SceneTree and wait until we have access to engine functionality
	SceneTree.Add(godot.NewCardGameUI().AsNode())
	startup.Scene() // starts up the scene and blocks until the engine shuts down.
}
