package auth

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

var usernameAdjectives = [...]string{
	"amber", "brisk", "calm", "clever", "cozy", "curious", "daring", "eager",
	"gentle", "golden", "happy", "jolly", "kind", "lively", "lucky", "mellow",
	"mighty", "nimble", "noble", "peppy", "playful", "proud", "quiet", "rapid",
	"radiant", "ready", "serene", "shiny", "sunny", "swift", "vibrant", "wise",
}

var usernameAnimals = [...]string{
	"badger", "beaver", "bison", "capybara", "cat", "cheetah", "crane", "dolphin",
	"eagle", "falcon", "fox", "gecko", "heron", "horse", "iguana", "jaguar",
	"koala", "lemur", "lion", "lynx", "meerkat", "otter", "owl", "panda",
	"parrot", "penguin", "rabbit", "raven", "seal", "tiger", "whale", "wolf",
}

func generateUsername() (string, error) {
	adjectiveIndex, err := randomUsernameIndex(len(usernameAdjectives))
	if err != nil {
		return "", fmt.Errorf("choose username adjective: %w", err)
	}

	animalIndex, err := randomUsernameIndex(len(usernameAnimals))
	if err != nil {
		return "", fmt.Errorf("choose username animal: %w", err)
	}

	return usernameAdjectives[adjectiveIndex] + "-" + usernameAnimals[animalIndex], nil
}

func randomUsernameIndex(length int) (int, error) {
	index, err := rand.Int(rand.Reader, big.NewInt(int64(length)))
	if err != nil {
		return 0, err
	}

	return int(index.Int64()), nil
}
