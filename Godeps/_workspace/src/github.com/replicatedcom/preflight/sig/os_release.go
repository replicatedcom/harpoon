// FIXME: this is straight outta docker machine
package sig

import (
	"bufio"
	"bytes"
	"fmt"
	"reflect"
	"strings"

	"github.com/replicatedcom/preflight/log"
)

type OSRelease struct {
	AnsiColor    string `osr:"ANSI_COLOR"`
	Name         string `osr:"NAME"`
	Version      string `osr:"VERSION"`
	ID           string `osr:"ID"`
	IDLike       string `osr:"ID_LIKE"`
	PrettyName   string `osr:"PRETTY_NAME"`
	VersionID    string `osr:"VERSION_ID"`
	HomeURL      string `osr:"HOME_URL"`
	SupportURL   string `osr:"SUPPORT_URL"`
	BugReportURL string `osr:"BUG_REPORT_URL"`
}

func NewOSRelease(contents []byte) (*OSRelease, error) {
	osr := &OSRelease{}
	if err := osr.ParseOSRelease(contents); err != nil {
		return nil, err
	}
	return osr, nil
}

func (osr *OSRelease) ParseOSRelease(osReleaseContents []byte) error {
	r := bytes.NewReader(osReleaseContents)
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		key, val, err := parseOSReleaseLine(scanner.Text())
		if err != nil {
			log.Warningf("Got an invalid line error parsing /etc/os-release: %v", err)
			continue
		}
		if err := osr.setIfPossible(key, val); err != nil {
			log.Debugf(err.Error())
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return nil
}

func (osr *OSRelease) setIfPossible(key, val string) error {
	v := reflect.ValueOf(osr).Elem()
	for i := 0; i < v.NumField(); i++ {
		fieldValue := v.Field(i)
		fieldType := v.Type().Field(i)
		originalName := fieldType.Tag.Get("osr")
		if key == originalName && fieldValue.Kind() == reflect.String {
			fieldValue.SetString(val)
			return nil
		}
	}
	return fmt.Errorf("Couldn't set key %s, no corresponding struct field found", key)
}

func parseOSReleaseLine(osrLine string) (string, string, error) {
	if osrLine == "" {
		return "", "", nil
	}

	vals := strings.Split(osrLine, "=")
	if len(vals) != 2 {
		return "", "", fmt.Errorf("Expected %s to split by '=' char into two strings, instead got %d strings", osrLine, len(vals))
	}
	key := vals[0]
	val := stripQuotes(vals[1])
	return key, val, nil
}

func stripQuotes(val string) string {
	if len(val) > 0 && val[0] == '"' {
		return val[1 : len(val)-1]
	}
	return val
}
