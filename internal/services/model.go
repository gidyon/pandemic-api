package services

import (
	"fmt"
	"github.com/jinzhu/gorm"
	"strings"
)

const defaultPageSize = 10

// NormalizePage parses page token and page size and reset weird values
func NormalizePage(pageToken, pageSize int32) (int, int) {
	if pageToken <= 0 {
		pageToken = 1
	}
	if pageSize <= 0 {
		pageSize = defaultPageSize
	}
	if pageSize > 20 {
		pageSize = 20
	}
	return int(pageToken), int(pageSize)
}

const indexName = "fts_search_index"

// CreateFullTextIndex creates a full-text index
func CreateFullTextIndex(db *gorm.DB, tableName string, columns ...string) error {
	if ok := db.Dialect().HasIndex(tableName, indexName); ok {
		return nil
	}
	sqlQuery := fmt.Sprintf(
		"CREATE FULLTEXT INDEX %s ON %s(%s)",
		indexName, tableName, strings.Join(columns, ","),
	)
	err := db.Table(tableName).Exec(sqlQuery).Error
	return err
}

// DropFullTextIndex drops a full-text index
func DropFullTextIndex(db *gorm.DB, tableName string) error {
	if ok := db.Dialect().HasIndex(tableName, indexName); !ok {
		return nil
	}
	sqlQuery := fmt.Sprintf("ALTER TABLE %s	DROP INDEX %s", tableName, indexName)
	err := db.Table(tableName).Exec(sqlQuery).Error
	return err
}

// ParseQuery parses a random query to a full-text query
func ParseQuery(query string, stopWords ...string) string {
	searchQueries := strings.Split(query, " ")
	parsedQueries := make([]string, 0, len(searchQueries))
	for _, queryToken := range searchQueries {
		if containStopWord(queryToken, stopWords) {
			continue
		}
		parsedQueries = append(parsedQueries, queryToken+"*")
	}
	return ">" + strings.Join(parsedQueries, " ")
}

func containStopWord(token string, stopWords []string) bool {
	for _, stopWord := range stopWords {
		if strings.ToLower(stopWord) == strings.ToLower(token) {
			return true
		}
	}
	return false
}
