package bootstrap

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	whisperRelease = "https://github.com/pink-tools/pink-whisper/releases/latest/download"
	modelURL       = "https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-large-v3.bin"
)

// GetWhisperDir returns the directory where pink-whisper is installed
func GetWhisperDir() string {
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, "pink-tools", "pink-whisper")
	os.MkdirAll(dir, 0755)
	return dir
}

func EnsureReady() error {
	whisperDir := GetWhisperDir()
	whisperPath := filepath.Join(whisperDir, whisperBinary())
	modelPath := filepath.Join(whisperDir, "ggml-large-v3.bin")

	// Check pink-whisper
	if _, err := os.Stat(whisperPath); os.IsNotExist(err) {
		fmt.Println("pink-whisper not found, downloading...")
		if err := downloadWhisper(whisperDir); err != nil {
			return fmt.Errorf("download pink-whisper: %w", err)
		}
		fmt.Println("pink-whisper ready")
	}

	// Check model
	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		fmt.Println("model not found, downloading (~3GB)...")
		if err := downloadFile(modelURL, modelPath); err != nil {
			return fmt.Errorf("download model: %w", err)
		}
		fmt.Println("model ready")
	}

	// Ensure executable
	os.Chmod(whisperPath, 0755)

	// macOS: remove quarantine
	if runtime.GOOS == "darwin" {
		exec.Command("xattr", "-d", "com.apple.quarantine", whisperPath).Run()
	}

	return nil
}

func whisperBinary() string {
	if runtime.GOOS == "windows" {
		return "pink-whisper.exe"
	}
	return "pink-whisper"
}

func downloadWhisper(whisperDir string) error {
	artifact := getArtifactName()
	url := whisperRelease + "/" + artifact

	// Download archive
	tmpFile := filepath.Join(whisperDir, artifact)
	if err := downloadFile(url, tmpFile); err != nil {
		return err
	}
	defer os.Remove(tmpFile)

	// Extract
	if strings.HasSuffix(artifact, ".zip") {
		return extractZip(tmpFile, whisperDir)
	}
	return extractTarGz(tmpFile, whisperDir)
}

func getArtifactName() string {
	switch {
	case runtime.GOOS == "darwin" && runtime.GOARCH == "arm64":
		return "darwin-arm64-coreml.tar.gz"
	case runtime.GOOS == "windows":
		if hasCUDA() {
			return "windows-amd64-cuda.zip"
		}
		return "windows-amd64-cpu.zip"
	case runtime.GOOS == "linux":
		if hasCUDA() {
			return "linux-amd64-cuda.tar.gz"
		}
		return "linux-amd64-cpu.tar.gz"
	default:
		return "linux-amd64-cpu.tar.gz"
	}
}

func hasCUDA() bool {
	_, err := exec.LookPath("nvidia-smi")
	return err == nil
}

func downloadFile(url, dest string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	// Progress tracking (every 5%)
	size := resp.ContentLength
	var written int64
	var lastPct int
	buf := make([]byte, 32*1024)
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			out.Write(buf[:n])
			written += int64(n)
			if size > 0 {
				pct := int(float64(written) / float64(size) * 100)
				if pct >= lastPct+5 || pct == 100 {
					fmt.Printf("%d%% (%.0f MB / %.0f MB)\n", pct, float64(written)/1024/1024, float64(size)/1024/1024)
					lastPct = pct
				}
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func extractTarGz(archive, dest string) error {
	f, err := os.Open(archive)
	if err != nil {
		return err
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		path := filepath.Join(dest, hdr.Name)

		switch hdr.Typeflag {
		case tar.TypeDir:
			os.MkdirAll(path, 0755)
		case tar.TypeReg:
			os.MkdirAll(filepath.Dir(path), 0755)
			out, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, os.FileMode(hdr.Mode))
			if err != nil {
				return err
			}
			io.Copy(out, tr)
			out.Close()
		}
	}
	return nil
}

func extractZip(archive, dest string) error {
	r, err := zip.OpenReader(archive)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		path := filepath.Join(dest, f.Name)

		if f.FileInfo().IsDir() {
			os.MkdirAll(path, 0755)
			continue
		}

		os.MkdirAll(filepath.Dir(path), 0755)

		rc, err := f.Open()
		if err != nil {
			return err
		}

		out, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, f.Mode())
		if err != nil {
			rc.Close()
			return err
		}

		io.Copy(out, rc)
		out.Close()
		rc.Close()
	}
	return nil
}
