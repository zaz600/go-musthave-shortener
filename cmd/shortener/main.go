package main

import (
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/zaz600/go-musthave-shortener/internal/app"
)

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnixMicro
	os.Exit(CLI(os.Args))
}

func CLI(args []string) int {
	if err := app.Run(args); err != nil {
		log.Error().Err(err).Msgf("Runtime error")
		return 1
	}
	return 0
}
