package input

import (
	"os"
	"path"
	"regexp"
	"strings"
)

func ParseInputSchema(src string) (string, error) {
	stat, err := os.Stat(src)
	if err != nil {
		return "", err
	}

	if stat.IsDir() {
		return parseMigrationFolder(src), nil
	}

	data, err := os.ReadFile(src)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func ParseInputQueries(src string) (string, error) {
	stat, err := os.Stat(src)
	if err != nil {
		return "", err
	}

	var queriesFilePath string
	if stat.IsDir() {
		queriesFilePath = path.Join(src, "queries.sql")
	} else {
		queriesFilePath = path.Join(path.Dir(src), "queries.sql")
	}

	if _, err = os.Stat(queriesFilePath); err == nil {
		data, err := os.ReadFile(queriesFilePath)
		if err != nil {
			return "", err
		}
		return string(data), nil
	}

	return "", nil
}

func parseMigrationFolder(folderName string) string {
	entries, err := os.ReadDir(folderName)
	if err != nil {
		panic(err)
	}

	sql := []string{}
	for _, entries := range entries {
		if entries.IsDir() {
			continue
		}

		fileName := path.Join(folderName, entries.Name())
		data, err := os.ReadFile(fileName)
		if err != nil {
			panic(err)
		}

		entry := parseMigrationFile(string(data))
		sql = append(sql, strings.TrimSpace(entry))
	}

	return strings.Join(sql, "\n")
}

var (
	parseGooseMetaUp   = regexp.MustCompile(`-- \+goose Up`)
	parseGooseMetaDown = regexp.MustCompile(`-- \+goose Down`)
)

func parseMigrationFile(content string) string {
	up := parseGooseMetaUp.FindStringIndex(content)
	down := parseGooseMetaDown.FindStringIndex(content)

	if len(up) == 0 {
		return ""
	}

	if len(down) == 0 {
		return content[up[1]:]
	}

	if up[0] < down[0] {
		return content[up[1]:down[0]]
	}

	return content[down[1]:up[0]]
}
