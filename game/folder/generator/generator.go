package generator

import "limeal.fr/launchygo/game/folder/shared"

// NOTE: A generator build the game

type Generator interface {
	Generate(debug bool, pCb shared.ProgressCallback)
}
