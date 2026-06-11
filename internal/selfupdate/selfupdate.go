// Package selfupdate обновляет бинарь tokenburning из GitHub Releases:
// проверяет последнюю версию, качает артефакт под текущую ОС/арх, сверяет SHA-256
// по checksums.txt и атомарно заменяет исполняемый файл. Сеть дёргается только при
// явном вызове (команда update или авто-апдейт демона) — не нарушает «по умолчанию
// ничего не шлёт в сеть».
package selfupdate

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const repoSlug = "rshatskiy/tokenburning"

// LatestTag возвращает тег последнего релиза (например, "v0.2.0").
func LatestTag() (string, error) {
	client := &http.Client{Timeout: 20 * time.Second}
	req, _ := http.NewRequest("GET", "https://api.github.com/repos/"+repoSlug+"/releases/latest", nil)
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("GitHub API: HTTP %d", resp.StatusCode)
	}
	var r struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return "", err
	}
	if r.TagName == "" {
		return "", fmt.Errorf("пустой tag_name")
	}
	return r.TagName, nil
}

// IsOlder сообщает, что current старее latest (по semver x.y.z). Непарсимая current (dev) → true.
func IsOlder(current, latest string) bool {
	a, b := parseVer(current), parseVer(latest)
	if a == nil {
		return true
	}
	if b == nil {
		return false
	}
	for i := 0; i < 3; i++ {
		if a[i] != b[i] {
			return a[i] < b[i]
		}
	}
	return false
}

func parseVer(s string) []int {
	s = strings.TrimPrefix(strings.TrimSpace(s), "v")
	s = strings.SplitN(s, "-", 2)[0]
	parts := strings.Split(s, ".")
	if len(parts) != 3 {
		return nil
	}
	out := make([]int, 3)
	for i := 0; i < 3; i++ {
		n, err := strconv.Atoi(parts[i])
		if err != nil {
			return nil
		}
		out[i] = n
	}
	return out
}

// DownloadAndApply качает релиз tag под текущую ОС/арх, сверяет SHA-256 и заменяет бинарь.
func DownloadAndApply(tag string) error {
	ver := strings.TrimPrefix(tag, "v")
	ext := "tar.gz"
	if runtime.GOOS == "windows" {
		ext = "zip"
	}
	fname := fmt.Sprintf("tokenburning_%s_%s_%s.%s", ver, runtime.GOOS, runtime.GOARCH, ext)
	archive, err := getAsset(tag, fname)
	if err != nil {
		return fmt.Errorf("скачивание %s: %w", fname, err)
	}
	sums, err := getAsset(tag, "checksums.txt")
	if err != nil {
		return fmt.Errorf("скачивание checksums.txt: %w", err)
	}
	want := checksumFor(string(sums), fname)
	if want == "" {
		return fmt.Errorf("контрольная сумма для %s не найдена", fname)
	}
	got := sha256.Sum256(archive)
	if !strings.EqualFold(want, hex.EncodeToString(got[:])) {
		return fmt.Errorf("контрольная сумма не совпала — обновление отменено")
	}
	bin, err := extractBinary(archive, ext == "zip")
	if err != nil {
		return err
	}
	if err := replaceExecutable(bin); err != nil {
		return fmt.Errorf("замена бинаря: %w (нужен доступ на запись; если ставили в системный каталог — sudo)", err)
	}
	return nil
}

// CheckAndApply: если current старее последнего — качает и ставит. (tag, updated, err).
func CheckAndApply(current string) (string, bool, error) {
	tag, err := LatestTag()
	if err != nil {
		return "", false, err
	}
	if !IsOlder(current, tag) {
		return tag, false, nil
	}
	if err := DownloadAndApply(tag); err != nil {
		return tag, false, err
	}
	return tag, true, nil
}

// getAsset качает релизный артефакт: зеркало на своём домене (GitHub
// release-CDN нестабилен в РФ) с одним повтором против разовых сетевых сбоев,
// затем — GitHub. В ошибке видны ВСЕ попытки, а не только последняя.
func getAsset(tag, fname string) ([]byte, error) {
	mirror := "https://tokenburning.ru/dl/" + tag
	attempts := []struct{ name, base string }{
		{"зеркало", mirror},
		{"зеркало, повтор", mirror},
		{"github", "https://github.com/" + repoSlug + "/releases/download/" + tag},
	}
	var errs []string
	for i, a := range attempts {
		if i == 1 {
			time.Sleep(2 * time.Second) // пауза перед повтором: переживаем мгновенные сбои
		}
		b, err := getBytes(a.base + "/" + fname)
		if err == nil {
			return b, nil
		}
		errs = append(errs, a.name+": "+err.Error())
	}
	return nil, fmt.Errorf("%s", strings.Join(errs, "; "))
}

func getBytes(url string) ([]byte, error) {
	client := &http.Client{Timeout: 90 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("%s: HTTP %d", url, resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 100<<20))
}

func checksumFor(sums, fname string) string {
	for _, line := range strings.Split(sums, "\n") {
		f := strings.Fields(line)
		if len(f) == 2 && f[1] == fname {
			return f[0]
		}
	}
	return ""
}

func extractBinary(archive []byte, isZip bool) ([]byte, error) {
	if isZip {
		zr, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
		if err != nil {
			return nil, err
		}
		for _, f := range zr.File {
			if filepath.Base(f.Name) == "tokenburning.exe" {
				rc, err := f.Open()
				if err != nil {
					return nil, err
				}
				defer rc.Close()
				return io.ReadAll(rc)
			}
		}
		return nil, fmt.Errorf("tokenburning.exe не найден в архиве")
	}
	gz, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		return nil, err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if filepath.Base(h.Name) == "tokenburning" {
			return io.ReadAll(tr)
		}
	}
	return nil, fmt.Errorf("бинарь tokenburning не найден в архиве")
}

// replaceExecutable атомарно заменяет текущий исполняемый файл новым содержимым.
func replaceExecutable(newBin []byte) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	return replaceFile(exe, newBin)
}

// replaceFile атомарно заменяет файл target новым содержимым (тестируемо).
func replaceFile(exe string, newBin []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(exe), ".tb-update-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(newBin); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Chmod(0o755); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}
	if runtime.GOOS == "windows" {
		old := exe + ".old"
		_ = os.Remove(old)
		if err := os.Rename(exe, old); err != nil {
			os.Remove(tmpName)
			return err
		}
		if err := os.Rename(tmpName, exe); err != nil {
			_ = os.Rename(old, exe)
			return err
		}
		_ = os.Remove(old)
		return nil
	}
	return os.Rename(tmpName, exe)
}
