package shared

import (
	"io"
	"log"
	"net/http"

	"limeal.fr/launchygo/game/folder/generator/manifests"
)

func DownloadArtifact(artifact *manifests.Artifact) ([]byte, error) {
	resp, err := http.Get(artifact.URL)
	if err != nil {
		log.Fatal("failed to download artifact")
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal("failed to read artifact")
	}

	return body, nil
}
