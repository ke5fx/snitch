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
	"strconv"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

const (
	repoOwner = "karol-broda"
	repoName  = "snitch"
	githubAPI = "https://api.github.com"
	firstUpgradeVersion = "0.1.8"
)

var (
	upgradeYes     bool
	upgradeVersion string
)

var upgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Check for updates and optionally upgrade snitch",
	Long: `Check for available updates and show upgrade instructions.

Use --yes to perform an in-place upgrade automatically.
Use --version to install a specific version.`,
	RunE: runUpgrade,
}

func init() {
	upgradeCmd.Flags().BoolVarP(&upgradeYes, "yes", "y", false, "Perform the upgrade automatically")
	upgradeCmd.Flags().StringVarP(&upgradeVersion, "version", "v", "", "Install a specific version (e.g., v0.1.7)")
	rootCmd.AddCommand(upgradeCmd)
}

type githubRelease struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
}

func runUpgrade(cmd *cobra.Command, args []string) error {
	current := Version

	if upgradeVersion != "" {
		return handleSpecificVersion(current, upgradeVersion)
	}

	latest, err := fetchLatestVersion()
	if err != nil {
		return fmt.Errorf("failed to check for updates: %w", err)
	}

	currentClean := strings.TrimPrefix(current, "v")
	latestClean := strings.TrimPrefix(latest, "v")

	printVersionComparison(current, latest)

	if currentClean == latestClean {
		green := color.New(color.FgGreen)
		green.Println("✓ you are running the latest version")
		return nil
	}

	if current == "dev" {
		yellow := color.New(color.FgYellow)
		yellow.Println("⚠ you are running a development build")
		fmt.Println()
		fmt.Println("use one of the methods below to install a release version:")
		fmt.Println()
		printUpgradeInstructions()
		return nil
	}

	green := color.New(color.FgGreen, color.Bold)
	green.Printf("✓ update available: %s → %s\n", current, latest)
	fmt.Println()

	if !upgradeYes {
		printUpgradeInstructions()
		fmt.Println()
		faint := color.New(color.Faint)
		cmdStyle := color.New(color.FgCyan)
		faint.Print("  in-place      ")
		cmdStyle.Println("snitch upgrade --yes")
		return nil
	}

	return performUpgrade(latest)
}

func handleSpecificVersion(current, target string) error {
	if !strings.HasPrefix(target, "v") {
		target = "v" + target
	}
	targetClean := strings.TrimPrefix(target, "v")

	printVersionComparisonTarget(current, target)

	if isVersionLower(targetClean, firstUpgradeVersion) {
		yellow := color.New(color.FgYellow)
		yellow.Printf("⚠ warning: the upgrade command was introduced in v%s\n", firstUpgradeVersion)
		faint := color.New(color.Faint)
		faint.Printf("  version %s does not include this command\n", target)
		faint.Println("  you will need to use other methods to upgrade from that version")
		fmt.Println()
	}

	currentClean := strings.TrimPrefix(current, "v")
	if currentClean == targetClean {
		green := color.New(color.FgGreen)
		green.Println("✓ you are already running this version")
		return nil
	}

	if !upgradeYes {
		faint := color.New(color.Faint)
		cmdStyle := color.New(color.FgCyan)
		if isVersionLower(targetClean, currentClean) {
			yellow := color.New(color.FgYellow)
			yellow.Printf("↓ this will downgrade from %s to %s\n", current, target)
		} else {
			green := color.New(color.FgGreen)
			green.Printf("↑ this will upgrade from %s to %s\n", current, target)
		}
		fmt.Println()
		faint.Print("run ")
		cmdStyle.Printf("snitch upgrade --version %s --yes", target)
		faint.Println(" to proceed")
		return nil
	}

	return performUpgrade(target)
}

func isVersionLower(v1, v2 string) bool {
	parts1 := parseVersion(v1)
	parts2 := parseVersion(v2)

	for i := 0; i < 3; i++ {
		if parts1[i] < parts2[i] {
			return true
		}
		if parts1[i] > parts2[i] {
			return false
		}
	}
	return false
}

func parseVersion(v string) [3]int {
	var parts [3]int
	segments := strings.Split(v, ".")

	for i := 0; i < len(segments) && i < 3; i++ {
		n, err := strconv.Atoi(segments[i])
		if err == nil {
			parts[i] = n
		}
	}
	return parts
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

func printVersionComparison(current, latest string) {
	faint := color.New(color.Faint)
	version := color.New(color.FgCyan)

	faint.Print("current  ")
	version.Println(current)
	faint.Print("latest   ")
	version.Println(latest)
	fmt.Println()
}

func printVersionComparisonTarget(current, target string) {
	faint := color.New(color.Faint)
	version := color.New(color.FgCyan)

	faint.Print("current  ")
	version.Println(current)
	faint.Print("target   ")
	version.Println(target)
	fmt.Println()
}

func printUpgradeInstructions() {
	bold := color.New(color.Bold)
	faint := color.New(color.Faint)
	cmd := color.New(color.FgCyan)

	bold.Println("upgrade options:")
	fmt.Println()

	faint.Print("  go install    ")
	cmd.Printf("go install github.com/%s/%s@latest\n", repoOwner, repoName)

	faint.Print("  shell script  ")
	cmd.Printf("curl -sSL https://raw.githubusercontent.com/%s/%s/master/install.sh | sh\n", repoOwner, repoName)

	faint.Print("  arch (aur)    ")
	cmd.Println("yay -S snitch-bin")

	faint.Print("  nix           ")
	cmd.Printf("nix profile upgrade --inputs-from github:%s/%s\n", repoOwner, repoName)
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

	faint := color.New(color.Faint)
	cyan := color.New(color.FgCyan)
	faint.Print("↓ downloading ")
	cyan.Printf("%s", archiveName)
	faint.Println("...")

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
		yellow := color.New(color.FgYellow)
		cmdStyle := color.New(color.FgCyan)

		yellow.Printf("⚠ elevated permissions required to install to %s\n", targetDir)
		fmt.Println()
		faint.Println("run with sudo or install to a user-writable location:")
		fmt.Println()
		faint.Print("  sudo         ")
		cmdStyle.Println("sudo snitch upgrade --yes")
		faint.Print("  custom dir   ")
		cmdStyle.Printf("curl -sSL https://raw.githubusercontent.com/%s/%s/master/install.sh | INSTALL_DIR=~/.local/bin sh\n",
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
		yellow := color.New(color.FgYellow)
		yellow.Fprintf(os.Stderr, "⚠ warning: failed to remove backup file %s: %v\n", backupPath, err)
	}

	green := color.New(color.FgGreen, color.Bold)
	green.Printf("✓ successfully upgraded to %s\n", version)
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

