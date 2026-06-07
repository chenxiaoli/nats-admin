package main

import "embed"

//go:embed frontend_dist/*
var frontendFS embed.FS
