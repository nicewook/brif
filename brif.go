package main

import (
	"log"
)

type Config struct {
	Version string
}
type Brif struct {
	Config *Config
}

func NewBrif() *Brif {

	log.Println("Create Brif")
	return &Brif{
		Config: &Config{Version: "v0.1.0"},
	}
}
