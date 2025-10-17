package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/database"
)

func (cfg *apiConfig) dbVideoToSignedVideo(video database.Video) (database.Video, error) {
	if video.VideoURL == nil {
		return video, nil
	}

	parts := strings.SplitN(*video.VideoURL, ",", 2)
	if len(parts) != 2 {
		log.Printf("bad video URL stored for id=%v: %q\n", video.ID, *video.VideoURL)
		return video, fmt.Errorf("could not split url properly: %v", parts)
	}
	bucket := strings.TrimSpace(parts[0])
	key := strings.TrimSpace(parts[1])

	signedUrl, err := generatePresignedURL(cfg.s3Client, bucket, key, 15*time.Minute)
	if err != nil {
		return video, fmt.Errorf("could not generate signed url: %w", err)
	}
	video.VideoURL = &signedUrl
	return video, nil
}

func processVideoForFastStart(filePath string) (string, error) {
	outPath := filePath + ".processing"

	cmd := exec.Command("ffmpeg", "-i", filePath, "-c", "copy", "-movflags", "faststart", "-f", "mp4", outPath)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("error running ffmpeg: %s, %w", stderr.String(), err)
	}

	fileInfo, err := os.Stat(outPath)
	if err != nil {
		return "", fmt.Errorf("could not stat processed file: %v", err)
	}
	if fileInfo.Size() == 0 {
		return "", fmt.Errorf("processed file is empty")
	}

	return outPath, nil
}

func getVideoAspectRatio(filePath string) (string, error) {
	data := &bytes.Buffer{}

	cmd := exec.Command("ffprobe", "-v", "error", "-print_format", "json", "-show_streams", filePath)
	cmd.Stdout = data
	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("error running ffprobe: %v", err)
	}

	var res FFVideoProbe
	err = json.Unmarshal(data.Bytes(), &res)
	if err != nil {
		return "", fmt.Errorf("error unmarshaling ffprobe output: %v", err)
	}

	width := res.Streams[0].Width
	height := res.Streams[0].Height
	if width == 0 || height == 0 {
		return "", fmt.Errorf("couldn't get width or height from ffprobe output")
	}

	return getAspectRatio(width, height)
}

func getAspectRatio(width, height int) (string, error) {
	if width == 0 || height == 0 {
		return "", fmt.Errorf("couldn't get width or height from ffprobe output")
	}

	ratio := float64(width) / float64(height)

	switch {
	case withTolerance(ratio, 16.0/9.0):
		return "16:9", nil
	case withTolerance(ratio, 9.0/16.0):
		return "9:16", nil
	default:
		return "other", nil
	}
}

func withTolerance(a, b float64) bool {
	const epsilon = 0.05
	return math.Abs(a-b) < epsilon
}

type FFVideoProbe struct {
	Streams []struct {
		Index              int    `json:"index"`
		CodecName          string `json:"codec_name,omitempty"`
		CodecLongName      string `json:"codec_long_name,omitempty"`
		Profile            string `json:"profile,omitempty"`
		CodecType          string `json:"codec_type"`
		CodecTagString     string `json:"codec_tag_string"`
		CodecTag           string `json:"codec_tag"`
		Width              int    `json:"width,omitempty"`
		Height             int    `json:"height,omitempty"`
		CodedWidth         int    `json:"coded_width,omitempty"`
		CodedHeight        int    `json:"coded_height,omitempty"`
		SampleAspectRatio  string `json:"sample_aspect_ratio,omitempty"`
		DisplayAspectRatio string `json:"display_aspect_ratio,omitempty"`
	} `json:"streams"`
}
