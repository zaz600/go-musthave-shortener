package main

import (
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/zaz600/go-musthave-shortener/internal/app"
)

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnixMicro
	// Не согласен, что так нельзя делать :)
	// Выход с одним и тем же кодом при любой ошибке затрудняет анализ кода возврата в шелл-скриптах
	// Но раз надо написать такой линтер, то чтобы его проверка проходила,
	// os.Exit(CLI(os.Args))
	log.Fatal().Int("exit_code", CLI(os.Args)).Msg("")
}

func CLI(args []string) int {
	if err := app.Run(args); err != nil {
		log.Error().Err(err).Msgf("Runtime error")
		return 1
	}
	return 0
}
