package storage

import (
	"embed"
	"fmt"
	"io/fs"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/sirupsen/logrus"
)

//go:embed queries/**/*.sql
var queriesFS embed.FS

// queryCache stores loaded SQL queries by their key
var queryCache = make(map[string]string)

// LoadQueries loads all SQL queries from the embedded filesystem
func LoadQueries() {
	_, file, line, _ := runtime.Caller(0)
	logrus.WithFields(logrus.Fields{
		"file": filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
		"line": line,
	}).Info("Loading SQL queries...")

	// Walk through the embedded filesystem
	err := fs.WalkDir(queriesFS, "queries", func(filePath string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if d.IsDir() {
			return nil
		}

		// Only process .sql files
		if !strings.HasSuffix(filePath, ".sql") {
			return nil
		}

		// Read the file content
		content, err := queriesFS.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to read query file %s: %w", filePath, err)
		}

		// Calculate the query key by removing the extension and "queries/" prefix
		// and replacing directory separators with dots
		key := strings.TrimSuffix(filePath, path.Ext(filePath))
		key = strings.TrimPrefix(key, "queries/")
		key = strings.ReplaceAll(key, "/", ".")

		// Store the query in the cache
		queryCache[key] = string(content)

		return nil
	})

	if err != nil {
		_, file, line, _ := runtime.Caller(0)
		logrus.WithFields(logrus.Fields{
			"file":  filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
			"line":  line,
			"error": err,
		}).Fatal("Failed to load SQL queries")
	}

	_, file, line, _ = runtime.Caller(0)
	logrus.WithFields(logrus.Fields{
		"file":  filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
		"line":  line,
		"count": len(queryCache),
	}).Info("Successfully loaded SQL queries")
}

// GetQuery returns a cached SQL query by its key
// For example: GetQuery("schema.create_users_table")
func GetQuery(key string) string {
	query, exists := queryCache[key]
	if !exists {
		_, file, line, _ := runtime.Caller(1)
		logrus.WithFields(logrus.Fields{
			"file": filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
			"line": line,
			"key":  key,
		}).Warn("Query not found")
		return ""
	}
	return query
}
