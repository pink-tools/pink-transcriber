package transcriber

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"os/exec"
	"strings"
)

const serverAddr = "127.0.0.1:7465"

func Transcribe(audioPath string) (string, error) {
	// Convert to PCM using ffmpeg
	pcmData, err := convertToPCM(audioPath)
	if err != nil {
		return "", fmt.Errorf("convert audio: %w", err)
	}

	// Send to pink-whisper
	text, err := sendToWhisper(pcmData)
	if err != nil {
		return "", fmt.Errorf("transcribe: %w", err)
	}

	return strings.TrimSpace(text), nil
}

func convertToPCM(audioPath string) ([]byte, error) {
	// ffmpeg -i input -ar 16000 -ac 1 -f s16le -
	cmd := exec.Command("ffmpeg",
		"-i", audioPath,
		"-ar", "16000",
		"-ac", "1",
		"-f", "s16le",
		"-loglevel", "error",
		"-",
	)

	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("ffmpeg: %s", string(exitErr.Stderr))
		}
		return nil, err
	}

	return output, nil
}

func sendToWhisper(pcmData []byte) (string, error) {
	conn, err := net.Dial("tcp", serverAddr)
	if err != nil {
		return "", fmt.Errorf("connect: %w (is pink-whisper running?)", err)
	}
	defer conn.Close()

	// Send size (4 bytes LE)
	sizeBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(sizeBytes, uint32(len(pcmData)))
	if _, err := conn.Write(sizeBytes); err != nil {
		return "", fmt.Errorf("send size: %w", err)
	}

	// Send PCM data
	if _, err := conn.Write(pcmData); err != nil {
		return "", fmt.Errorf("send audio: %w", err)
	}

	// Receive response size
	respSizeBytes := make([]byte, 4)
	if _, err := io.ReadFull(conn, respSizeBytes); err != nil {
		return "", fmt.Errorf("read response size: %w", err)
	}
	respSize := binary.LittleEndian.Uint32(respSizeBytes)

	// Receive text
	textBytes := make([]byte, respSize)
	if _, err := io.ReadFull(conn, textBytes); err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	return string(textBytes), nil
}

func CheckFFmpeg() error {
	_, err := exec.LookPath("ffmpeg")
	if err != nil {
		return fmt.Errorf("ffmpeg not found in PATH")
	}
	return nil
}

func IsServerRunning() bool {
	conn, err := net.Dial("tcp", serverAddr)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}
