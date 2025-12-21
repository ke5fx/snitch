package cmd

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
)

const (
	repoOwner = "karol-broda"
	repoName  = "snitch"
	githubAPI = "https://api.github.com"
)

var upgradeYes bool

var upgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Check for updates and optionally upgrade snitch",
	Long: `Check for available updates and show upgrade instructions.

Use --yes to perform an in-place upgrade automatically.`,
	RunE: runUpgrade,
}

func init() {
	upgradeCmd.Flags().BoolVarP(&upgradeYes, "yes", "y", false, "Perform the upgrade automatically")
	rootCmd.AddCommand(upgradeCmd)
}

type githubRelease struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
}

func runUpgrade(cmd *cobra.Command, args []string) error {
	current := Version
	latest, err := fetchLatestVersion()
	if err != nil {
		return fmt.Errorf("failed to check for updates: %w", err)
	}

	currentClean := strings.TrimPrefix(current, "v")
	latestClean := strings.TrimPrefix(latest, "v")

	fmt.Printf("current: %s\n", current)
	fmt.Printf("latest:  %s\n", latest)
	fmt.Println()

	if currentClean == latestClean {
		fmt.Println("you are running the latest version")
		return nil
	}

	if current == "dev" {
		fmt.Println("you are running a development build")
		fmt.Println("use one of the methods below to install a release version:")
		printUpgradeInstructions()
		return nil
	}

	fmt.Printf("update available: %s -> %s\n", current, latest)
	fmt.Println()

	if !upgradeYes {
		printUpgradeInstructions()
		fmt.Println()
		fmt.Println("or run 'snitch upgrade --yes' to upgrade in-place")
		return nil
	}

	return performUpgrade(latest)
}

func fetchLatestVersion() (string, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/releases/latest", githubAPI, repoOwner, repoName)

	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("github api returned status %d", resp.StatusCode)
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", err
	}

	if release.TagName == "" {
		return "", fmt.Errorf("no releases found")
	}

	return release.TagName, nil
}

func printUpgradeInstructions() {
	fmt.Println("upgrade options:")
	fmt.Println()
	fmt.Println("  go install:")
	fmt.Printf("    go install github.com/%s/%s@latest\n", repoOwner, repoName)
	fmt.Println()
	fmt.Println("  shell script:")
	fmt.Printf("    curl -sSL https://raw.githubusercontent.com/%s/%s/master/install.sh | sh\n", repoOwner, repoName)
	fmt.Println()
	fmt.Println("  arch linux (aur):")
	fmt.Println("    yay -S snitch-bin")
	fmt.Println()
	fmt.Println("  nix:")
	fmt.Printf("    nix profile upgrade --inputs-from github:%s/%s\n", repoOwner, repoName)
}

func performUpgrade(version string) error {
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return fmt.Errorf("failed to resolve executable path: %w", err)
	}

	goos := runtime.GOOS
	goarch := runtime.GOARCH

	versionClean := strings.TrimPrefix(version, "v")
	archiveName := fmt.Sprintf("%s_%s_%s_%s.tar.gz", repoName, versionClean, goos, goarch)
	downloadURL := fmt.Sprintf("https://github.com/%s/%s/releases/download/%s/%s",
		repoOwner, repoName, version, archiveName)

	fmt.Printf("downloading %s...\n", archiveName)

	resp, err := http.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("failed to download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	tmpDir, err := os.MkdirTemp("", "snitch-upgrade-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	binaryPath, err := extractBinaryFromTarGz(resp.Body, tmpDir)
	if err != nil {
		return fmt.Errorf("failed to extract binary: %w", err)
	}

	// check if we can write to the target location
	targetDir := filepath.Dir(execPath)
	if !isWritable(targetDir) {
		fmt.Printf("elevated permissions required to install to %s\n", targetDir)
		fmt.Println()
		fmt.Println("run with sudo or install to a user-writable location:")
		fmt.Printf("  sudo snitch upgrade --yes\n")
		fmt.Println()
		fmt.Println("or use the install script with a custom directory:")
		fmt.Printf("  curl -sSL https://raw.githubusercontent.com/%s/%s/master/install.sh | INSTALL_DIR=~/.local/bin sh\n",
			repoOwner, repoName)
		return nil
	}

	// replace the binary
	backupPath := execPath + ".bak"
	if err := os.Rename(execPath, backupPath); err != nil {
		return fmt.Errorf("failed to backup current binary: %w", err)
	}

	if err := copyFile(binaryPath, execPath); err != nil {
		// try to restore backup
		if restoreErr := os.Rename(backupPath, execPath); restoreErr != nil {
			return fmt.Errorf("failed to install new binary and restore backup: %w (restore error: %v)", err, restoreErr)
		}
		return fmt.Errorf("failed to install new binary: %w", err)
	}

	if err := os.Chmod(execPath, 0755); err != nil {
		return fmt.Errorf("failed to set executable permissions: %w", err)
	}

	if err := os.Remove(backupPath); err != nil {
		// non-fatal, just warn
		fmt.Fprintf(os.Stderr, "warning: failed to remove backup file %s: %v\n", backupPath, err)
	}

	fmt.Printf("successfully upgraded to %s\n", version)
	return nil
}

func extractBinaryFromTarGz(r io.Reader, destDir string) (string, error) {
	gzr, err := gzip.NewReader(r)
	if err != nil {
		return "", err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}

		if header.Typeflag != tar.TypeReg {
			continue
		}

		// look for the snitch binary
		name := filepath.Base(header.Name)
		if name != repoName {
			continue
		}

		destPath := filepath.Join(destDir, name)
		outFile, err := os.Create(destPath)
		if err != nil {
			return "", err
		}

		if _, err := io.Copy(outFile, tr); err != nil {
			outFile.Close()
			return "", err
		}
		outFile.Close()

		return destPath, nil
	}

	return "", fmt.Errorf("binary not found in archive")
}

func isWritable(path string) bool {
	testFile := filepath.Join(path, ".snitch-write-test")
	f, err := os.Create(testFile)
	if err != nil {
		return false
	}
	f.Close()
	os.Remove(testFile)
	return true
}

func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return err
	}

	return dstFile.Sync()
}

