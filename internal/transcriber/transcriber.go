package transcriber

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

const (
	chunkSec       = 120
	overlapSec     = 5
	chunkThreshold = 120

	defaultLocalAddr  = "localhost:7465"
	defaultRemoteAddr = "transcribe.pinkhaired.com:7465"
)

func getLocalAddr() string {
	if addr := os.Getenv("WHISPER_LOCAL_ADDR"); addr != "" {
		return addr
	}
	return defaultLocalAddr
}

func getRemoteAddr() string {
	if addr := os.Getenv("WHISPER_REMOTE_ADDR"); addr != "" {
		return addr
	}
	return defaultRemoteAddr
}

func selectBackend() string {
	localAddr := getLocalAddr()
	fmt.Fprintf(os.Stderr, "[debug] trying local: %s\n", localAddr)
	conn, err := net.Dial("tcp", localAddr)
	if err == nil {
		conn.Close()
		fmt.Fprintf(os.Stderr, "[debug] using local\n")
		return localAddr
	}
	fmt.Fprintf(os.Stderr, "[debug] local failed: %v\n", err)
	remoteAddr := getRemoteAddr()
	fmt.Fprintf(os.Stderr, "[debug] using remote: %s\n", remoteAddr)
	return remoteAddr
}

func Transcribe(audioPath string) (string, error) {
	fmt.Fprintf(os.Stderr, "[debug] getting duration...\n")
	duration, err := getAudioDuration(audioPath)
	if err != nil {
		return "", fmt.Errorf("get duration: %w", err)
	}
	fmt.Fprintf(os.Stderr, "[debug] duration: %.2fs\n", duration)

	backend := selectBackend()
	fmt.Fprintf(os.Stderr, "[debug] backend selected: %s\n", backend)

	if duration <= chunkThreshold {
		return transcribeSingle(audioPath, backend)
	}

	return transcribeChunked(audioPath, duration, backend)
}

func transcribeSingle(audioPath, backend string) (string, error) {
	pcmData, err := convertToPCM(audioPath, 0, 0)
	if err != nil {
		return "", fmt.Errorf("convert audio: %w", err)
	}

	text, err := sendToWhisper(pcmData, backend)
	if err != nil {
		return "", fmt.Errorf("transcribe: %w", err)
	}

	return strings.TrimSpace(text), nil
}

func transcribeChunked(audioPath string, duration float64, backend string) (string, error) {
	var results []string
	start := 0.0

	for start < duration {
		chunkDuration := float64(chunkSec)
		if start+chunkDuration > duration {
			chunkDuration = duration - start
		}

		pcmData, err := convertToPCM(audioPath, start, chunkDuration)
		if err != nil {
			return "", fmt.Errorf("convert chunk at %.0fs: %w", start, err)
		}

		text, err := sendToWhisper(pcmData, backend)
		if err != nil {
			return "", fmt.Errorf("transcribe chunk at %.0fs: %w", start, err)
		}

		results = append(results, strings.TrimSpace(text))
		start += float64(chunkSec - overlapSec)
	}

	return strings.Join(results, " "), nil
}

func getAudioDuration(audioPath string) (float64, error) {
	cmd := exec.Command("ffprobe",
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		audioPath,
	)

	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return 0, fmt.Errorf("ffprobe: %s", string(exitErr.Stderr))
		}
		return 0, err
	}

	duration, err := strconv.ParseFloat(strings.TrimSpace(string(output)), 64)
	if err != nil {
		return 0, fmt.Errorf("parse duration: %w", err)
	}

	return duration, nil
}

func convertToPCM(audioPath string, startSec, durationSec float64) ([]byte, error) {
	args := []string{}

	if startSec > 0 {
		args = append(args, "-ss", fmt.Sprintf("%.2f", startSec))
	}

	args = append(args, "-i", audioPath)

	if durationSec > 0 {
		args = append(args, "-t", fmt.Sprintf("%.2f", durationSec))
	}

	args = append(args, "-ar", "16000", "-ac", "1", "-f", "s16le", "-loglevel", "error", "-")

	cmd := exec.Command("ffmpeg", args...)
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("ffmpeg: %s", string(exitErr.Stderr))
		}
		return nil, err
	}

	return output, nil
}

func sendToWhisper(pcmData []byte, addr string) (string, error) {
	fmt.Fprintf(os.Stderr, "[debug] connecting to %s...\n", addr)
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return "", fmt.Errorf("connect to %s: %w", addr, err)
	}
	defer conn.Close()
	fmt.Fprintf(os.Stderr, "[debug] connected, sending %d bytes...\n", len(pcmData))

	sizeBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(sizeBytes, uint32(len(pcmData)))
	if _, err := conn.Write(sizeBytes); err != nil {
		return "", fmt.Errorf("send size: %w", err)
	}

	if _, err := conn.Write(pcmData); err != nil {
		return "", fmt.Errorf("send audio: %w", err)
	}
	fmt.Fprintf(os.Stderr, "[debug] sent, waiting for response...\n")

	respSizeBytes := make([]byte, 4)
	if _, err := io.ReadFull(conn, respSizeBytes); err != nil {
		return "", fmt.Errorf("read response size: %w", err)
	}
	respSize := binary.LittleEndian.Uint32(respSizeBytes)
	fmt.Fprintf(os.Stderr, "[debug] response size: %d bytes\n", respSize)

	textBytes := make([]byte, respSize)
	if _, err := io.ReadFull(conn, textBytes); err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}
	fmt.Fprintf(os.Stderr, "[debug] done\n")

	return string(textBytes), nil
}

func CheckFFmpeg() error {
	_, err := exec.LookPath("ffmpeg")
	if err != nil {
		return fmt.Errorf("ffmpeg not found in PATH")
	}
	return nil
}
