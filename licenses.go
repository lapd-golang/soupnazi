package soupnazi

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"runtime"
	"strings"

	"github.com/Sirupsen/logrus"

	"github.com/mitchellh/go-homedir"
)

// If this environment variable is set, we use the file it
// points to as the name of the user's settings file
var envvar = "SOUPNAZI_CONFIG_FILE"

func LicenseFile() string {
	// Look to see if SOUPNAZI_CONFIG_FILE is set.  If so, read the
	// file it points to.
	if os.Getenv(envvar) != "" {
		return os.Getenv(envvar)
	}

	home, _ := homedir.Dir()

	// Otherwise, find out what platform we are on...
	platform := runtime.GOOS

	datadir := ""

	switch platform {
	case "windows":
		// On windows, check to see if APPDATA is defined...
		datadir = os.Getenv("APPDATA")
		if datadir == "" {
			datadir = path.Join(home, ".config")
		}
	case "linux":
		// On windows, check to see if APPDATA is defined...
		datadir = os.Getenv("XDG_CONFIG_HOME")
		if datadir == "" {
			datadir = path.Join(home, ".config")
		}
	case "darwin":
		datadir = path.Join(home, "Library", "Preferences")
	default:
		log.Printf("Unknown platform %v", platform)
	}

	return path.Join(datadir, "soupnazi", "licenses")
}

// ParseLicenses returns an error if the license
// file is corrupted
func ParseLicenses(lfile string, stream *logrus.Logger) ([]string, error) {
	ret := []string{}
	blank := []string{}

	// If it exists, check that the last
	f, err := os.Open(lfile)
	if err != nil {
		return ret, fmt.Errorf("Error trying to open license file %s: %v", lfile, err)
	}
	defer f.Close()

	stream.Infof("Extracting licenses from %s", lfile)

	reader := bufio.NewReader(f)

	process := func(r io.Reader) (string, error) {
		line, err := reader.ReadString('\n')
		if err != nil {
			return "", err
		}
		stream.Infof("  Raw line: '%s'", line)
		t := strings.Trim(line, "\n")
		stream.Infof("  Trimmed line: '%s'", t)
		return t, nil
	}

	line, err := process(reader)
	if err == io.EOF {
		return ret, nil
	}
	if err != nil {
		return blank, err
	}
	if line != "" {
		ret = append(ret, line)
	}
	for err == nil {
		line, err := process(reader)
		if err == io.EOF {
			return ret, nil
		}
		if err != nil {
			return blank, err
		}
		if line != "" {
			ret = append(ret, line)
		}
	}
	return blank, fmt.Errorf("License file corrupted, ended with %s", line)
}

func AddLicense(lic string, stream *logrus.Logger) error {
	token := RawToken(lic)

	if token == nil {
		return fmt.Errorf("License %s is not a valid JWT", lic)
	} else {
		stream.Debugf("License passed syntax checking: %v", token)
	}

	lfile := LicenseFile()
	if lfile == "" {
		return fmt.Errorf("Unable to identify settings file")
	}

	stream.Infof("License file location: %s", lfile)

	// If file doesn't exist, "touch" it
	if _, err := os.Stat(lfile); os.IsNotExist(err) {
		pdir := path.Dir(lfile)
		if pdir == "" {
			return fmt.Errorf("Could't determine parent directory of %s: %v", lfile, err)
		}

		stream.Infof("  Creating parent directories")
		err := os.MkdirAll(pdir, 0777)
		if err != nil {
			return fmt.Errorf("Could't create parent directory %s: %v", pdir, err)
		}

		stream.Infof("  Creating license file")
		f, err := os.Create(lfile)
		if err != nil {
			return fmt.Errorf("Error trying create new licenses file at %s: %v", lfile, err)
		}
		f.Close()
	}

	lics, err := ParseLicenses(lfile, stream)
	if err != nil {
		return fmt.Errorf("License file at %s is corrupted: %v")
	}

	// Check for duplicates
	for _, l := range lics {
		if lic == l {
			stream.Infof("  Not adding '%s' because it is a duplicate of an existing entry", l)
			return nil
		}
	}

	f, err := os.OpenFile(lfile, os.O_RDWR, 0777)
	if err != nil {
		return fmt.Errorf("Error trying to open license file %s: %v", lfile, err)
	}

	_, err = f.Seek(0, 2)
	if err != nil {
		return fmt.Errorf("Error seeking end of license file %s: %v", lfile, err)
	}

	_, err = f.Write([]byte(fmt.Sprintf("%s\n", lic)))
	if err != nil {
		return fmt.Errorf("Error writing license to %s: %v", lfile, err)
	}

	return nil
}
