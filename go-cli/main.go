package main

import (
    "fmt"
    "os"
)

func main() {
    cfg, err := resolveConfig(os.Args[1:])
    if err != nil {
        fmt.Fprintln(os.Stderr, "simple-cli error:", err)
        os.Exit(1)
    }

    renderer := NewRenderer(cfg)
    client, err := newSimpleClient(cfg, renderer)
    if err != nil {
        fmt.Fprintln(os.Stderr, "simple-cli error:", err)
        os.Exit(1)
    }
    defer client.Close()

    if err := runRepl(&cfg, client, renderer); err != nil {
        fmt.Fprintln(os.Stderr, "simple-cli error:", err)
        os.Exit(1)
    }
}
