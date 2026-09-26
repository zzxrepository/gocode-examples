package main

import "fmt"

const extraUsage = "(no extra commands)"

func extra(cmd string, o options) error { return fmt.Errorf("unknown command: %s", cmd) }
