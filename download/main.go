package main

import (
	"os"
	"os/exec"
	"strings"
)

func downloadYTDLP(
	audio_format string,
	track string,
	output_path string,
) {
	cmd := exec.Command(
		"yt-dlp",
		"-x",
		"--audio-format", audio_format,
		"-o", output_path,
		track,
	)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		os.Exit(1)
	}
}

func main() {
	machine_path, _ := os.Getwd()

	tracks_src_file := machine_path + "/test_samples/net/tracks.txt"

	content, err := os.ReadFile(tracks_src_file)
	if err != nil {
		panic(err)
	}

	tracks := string(content)
	tracks_list := strings.Split(tracks, "\n")

	for _, track := range tracks_list {
		track = track[:len(track)-1] // Remove newline character
		chunks := strings.Split(track, "=")
		track_id := chunks[1] // Extract track ID
		audio_format := "wav"
		output_path := machine_path + "/test_samples/net/" + track_id

		downloadYTDLP(audio_format, track, output_path)
	}
}
