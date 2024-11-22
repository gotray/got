package install

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/gotray/got/internal/env"
)

const (
	baseURL = "https://github.com/indygreg/python-build-standalone/releases/download/%s"
)

type pythonBuild struct {
	arch     string
	os       string
	variant  string
	debug    bool
	shared   bool
	fullPack bool
}

// getPythonURL returns the appropriate Python standalone URL for the current platform
func getPythonURL(version, buildDate, arch, os string, freeThreaded, debug bool) string {
	// Map GOARCH to Python build architecture
	archMap := map[string]string{
		"amd64": "x86_64",
		"arm64": "aarch64",
		"386":   "i686",
	}

	pythonArch, ok := archMap[arch]
	if !ok {
		return ""
	}

	build := pythonBuild{
		arch:     pythonArch,
		fullPack: true,
		debug:    debug,
	}

	switch os {
	case "darwin":
		build.os = "apple-darwin"
		if freeThreaded {
			build.variant = "freethreaded"
			if build.debug {
				build.variant += "+debug"
			} else {
				build.variant += "+pgo"
			}
		} else {
			if build.debug {
				build.variant = "debug"
			} else {
				build.variant = "pgo"
			}
		}
	case "linux":
		build.os = "unknown-linux-gnu"
		if freeThreaded {
			build.variant = "freethreaded"
			if build.debug {
				build.variant += "+debug"
			} else {
				build.variant += "+pgo"
			}
		} else {
			if build.debug {
				build.variant = "debug"
			} else {
				build.variant = "pgo"
			}
		}
	case "windows":
		build.os = "pc-windows-msvc"
		build.shared = true
		if freeThreaded {
			build.variant = "freethreaded+pgo"
		} else {
			build.variant = "pgo"
		}
	default:
		return ""
	}

	// Construct filename
	filename := fmt.Sprintf("cpython-%s+%s-%s-%s", version, buildDate, build.arch, build.os)
	if build.shared {
		filename += "-shared"
	}
	filename += "-" + build.variant
	if build.fullPack {
		filename += "-full"
	}
	filename += ".tar.zst"

	return fmt.Sprintf(baseURL, buildDate) + "/" + filename
}

// updateMacOSDylibs updates the install names of dylib files on macOS
func updateMacOSDylibs(pythonDir string, verbose bool) error {
	libDir := filepath.Join(pythonDir, "lib")
	entries, err := os.ReadDir(libDir)
	if err != nil {
		return fmt.Errorf("failed to read lib directory: %v", err)
	}

	absLibDir, err := filepath.Abs(libDir)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %v", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".dylib") {
			dylibPath := filepath.Join(libDir, entry.Name())
			if verbose {
				fmt.Printf("Updating install name for: %s\n", dylibPath)
			}

			// Get the current install name
			cmd := exec.Command("otool", "-D", dylibPath)
			output, err := cmd.Output()
			if err != nil {
				return fmt.Errorf("failed to get install name for %s: %v", dylibPath, err)
			}

			// Parse the output to get the current install name
			lines := strings.Split(string(output), "\n")
			if len(lines) < 2 {
				continue
			}
			currentName := strings.TrimSpace(lines[1])
			if currentName == "" {
				continue
			}

			// Calculate new install name using absolute path
			newName := filepath.Join(absLibDir, filepath.Base(currentName))

			fmt.Printf("Updating install name for %s to %s\n", dylibPath, newName)
			// Update the install name
			cmd = exec.Command("install_name_tool", "-id", newName, dylibPath)
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("failed to update install name for %s: %v", dylibPath, err)
			}
		}
	}
	return nil
}

// genWinPyPkgConfig generates pkg-config files for Windows
func genWinPyPkgConfig(pythonRoot, pkgConfigDir string) error {
	fmt.Printf("Generating pkg-config files in %s\n", pkgConfigDir)
	if err := os.MkdirAll(pkgConfigDir, 0755); err != nil {
		return fmt.Errorf("failed to create pkgconfig directory: %v", err)
	}

	// Get Python environment
	pyEnv := env.NewPythonEnv(pythonRoot)
	pythonBin, err := pyEnv.Python()
	if err != nil {
		return fmt.Errorf("failed to get Python executable: %v", err)
	}

	// Get Python version and check if freethreaded
	cmd := exec.Command(pythonBin, "-c", `
import sys
import sysconfig
version = f'{sys.version_info.major}.{sys.version_info.minor}'
is_freethreaded = hasattr(sys, "gettotalrefcount")
print(f'{version}\n{is_freethreaded}')
`)
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get Python info: %v", err)
	}

	info := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(info) != 2 {
		return fmt.Errorf("unexpected Python info output format")
	}

	version := strings.TrimSpace(info[0])
	isFreethreaded := strings.TrimSpace(info[1]) == "True"

	// Prepare version-specific library names
	versionNoPoints := strings.ReplaceAll(version, ".", "")
	libSuffix := ""
	if isFreethreaded {
		libSuffix = "t"
	}

	// Template for the pkg-config files
	embedTemplate := `prefix=${pcfiledir}/../..
exec_prefix=${prefix}
libdir=${exec_prefix}
includedir=${prefix}/include

Name: Python
Description: Embed Python into an application
Requires:
Version: %s
Libs.private:
Libs: -L${libdir} -lpython%s%s
Cflags: -I${includedir}
`

	normalTemplate := `prefix=${pcfiledir}/../..
exec_prefix=${prefix}
libdir=${exec_prefix}
includedir=${prefix}/include

Name: Python
Description: Python library
Requires:
Version: %s
Libs.private:
Libs: -L${libdir} -lpython3%s
Cflags: -I${includedir}
`

	// Generate file pairs
	filePairs := []struct {
		name     string
		template string
		embed    bool
	}{
		{fmt.Sprintf("python-%s%s.pc", version, libSuffix), normalTemplate, false},
		{fmt.Sprintf("python-%s%s-embed.pc", version, libSuffix), embedTemplate, true},
		{"python3" + libSuffix + ".pc", normalTemplate, false},
		{"python3" + libSuffix + "-embed.pc", embedTemplate, true},
	}

	// If freethreaded, also generate non-t versions with the same content
	if isFreethreaded {
		additionalPairs := []struct {
			name     string
			template string
			embed    bool
		}{
			{fmt.Sprintf("python-%s.pc", version), normalTemplate, false},
			{fmt.Sprintf("python-%s-embed.pc", version), embedTemplate, true},
			{"python3.pc", normalTemplate, false},
			{"python3-embed.pc", embedTemplate, true},
		}
		filePairs = append(filePairs, additionalPairs...)
	}

	// Write all pkg-config files
	for _, pair := range filePairs {
		pcPath := filepath.Join(pkgConfigDir, pair.name)
		var content string
		if pair.embed {
			content = fmt.Sprintf(pair.template, version, versionNoPoints, libSuffix)
		} else {
			content = fmt.Sprintf(pair.template, version, libSuffix)
		}
		if err := os.WriteFile(pcPath, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write %s: %v", pair.name, err)
		}
	}

	return nil
}

// updatePkgConfig updates the prefix in pkg-config files to use absolute path
func updatePkgConfig(projectPath string) error {
	pythonPath := env.GetPythonRoot(projectPath)
	pkgConfigDir := env.GetPythonPkgConfigDir(projectPath)

	entries, err := os.ReadDir(pkgConfigDir)
	if err != nil {
		return fmt.Errorf("failed to read pkgconfig directory: %v", err)
	}

	absPath, err := filepath.Abs(pythonPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %v", err)
	}

	// Helper function to write a .pc file with the correct prefix
	writePC := func(path string, content []byte) error {
		newContent := strings.ReplaceAll(string(content), "prefix=/install", "prefix="+absPath)
		return os.WriteFile(path, []byte(newContent), 0644)
	}

	// Regular expressions for matching file patterns
	normalPattern := regexp.MustCompile(`^python-(\d+\.\d+)t?\.pc$`)
	embedPattern := regexp.MustCompile(`^python-(\d+\.\d+)t?-embed\.pc$`)

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".pc") {
			pcFile := filepath.Join(pkgConfigDir, entry.Name())

			// Read file content
			content, err := os.ReadFile(pcFile)
			if err != nil {
				return fmt.Errorf("failed to read %s: %v", pcFile, err)
			}

			// Update original file
			if err := writePC(pcFile, content); err != nil {
				return fmt.Errorf("failed to update %s: %v", pcFile, err)
			}

			name := entry.Name()
			// Create additional copies based on patterns
			copies := make(map[string]bool)

			// Handle python-X.YZt.pc and python-X.YZ.pc patterns
			if matches := normalPattern.FindStringSubmatch(name); matches != nil {
				if strings.Contains(name, "t.pc") {
					// python-3.13t.pc -> python3.pc and python3t.pc
					copies["python3t.pc"] = true
					copies["python3.pc"] = true
					// Also create non-t version
					noT := fmt.Sprintf("python-%s.pc", matches[1])
					if err := writePC(filepath.Join(pkgConfigDir, noT), content); err != nil {
						return fmt.Errorf("failed to write %s: %v", noT, err)
					}
				} else {
					// python-3.13.pc -> python3.pc
					copies["python3.pc"] = true
				}
			}

			// Handle python-X.YZt-embed.pc and python-X.YZ-embed.pc patterns
			if matches := embedPattern.FindStringSubmatch(name); matches != nil {
				if strings.Contains(name, "t-embed.pc") {
					// python-3.13t-embed.pc -> python3-embed.pc and python3t-embed.pc
					copies["python3t-embed.pc"] = true
					copies["python3-embed.pc"] = true
					// Also create non-t version
					noT := fmt.Sprintf("python-%s-embed.pc", matches[1])
					if err := writePC(filepath.Join(pkgConfigDir, noT), content); err != nil {
						return fmt.Errorf("failed to write %s: %v", noT, err)
					}
				} else {
					// python-3.13-embed.pc -> python3-embed.pc
					copies["python3-embed.pc"] = true
				}
			}

			// Write all unique copies
			for copyName := range copies {
				copyPath := filepath.Join(pkgConfigDir, copyName)
				if err := writePC(copyPath, content); err != nil {
					return fmt.Errorf("failed to write %s: %v", copyPath, err)
				}
			}
		}
	}
	return nil
}

// installPythonEnv downloads and installs Python standalone build
func installPythonEnv(projectPath string, version, buildDate string, freeThreaded, debug bool, verbose bool) error {
	fmt.Printf("Installing Python %s in %s\n", version, projectPath)
	pythonRoot := env.GetPythonRoot(projectPath)

	// Remove existing Python directory if it exists
	if err := os.RemoveAll(pythonRoot); err != nil {
		return fmt.Errorf("error removing existing Python directory: %v", err)
	}

	// Get Python URL
	url := getPythonURL(version, buildDate, runtime.GOARCH, runtime.GOOS, freeThreaded, debug)
	if url == "" {
		return fmt.Errorf("unsupported platform")
	}

	if err := downloadAndExtract("Python", version, url, pythonRoot, "python/install", verbose); err != nil {
		return fmt.Errorf("error downloading and extracting Python: %v", err)
	}

	// After extraction, update dylib install names on macOS
	if runtime.GOOS == "darwin" {
		if err := updateMacOSDylibs(pythonRoot, verbose); err != nil {
			return fmt.Errorf("error updating dylib install names: %v", err)
		}
	}

	if runtime.GOOS == "windows" {
		pkgConfigDir := env.GetPythonPkgConfigDir(projectPath)
		if err := genWinPyPkgConfig(pythonRoot, pkgConfigDir); err != nil {
			return err
		}
	}

	if err := updatePkgConfig(projectPath); err != nil {
		return fmt.Errorf("error updating pkg-config: %v", err)
	}

	// Create Python environment
	pyEnv := env.NewPythonEnv(pythonRoot)

	if verbose {
		fmt.Println("Installing Python dependencies...")
	}

	if err := pyEnv.RunPip("install", "--upgrade", "pip", "setuptools", "wheel"); err != nil {
		return fmt.Errorf("error upgrading pip, setuptools, whell")
	}

	pythonPath, err := pyEnv.GetPythonPath()
	if err != nil {
		return fmt.Errorf("failed to get Python path: %v", err)
	}
	// Write environment variables to env.txt
	if err := env.WriteEnvFile(projectPath, pythonRoot, pythonPath); err != nil {
		return fmt.Errorf("error writing environment file: %v", err)
	}

	return nil
}
