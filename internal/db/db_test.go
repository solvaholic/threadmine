package db

import (
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestEmbeddedSchemaFTS5(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "threadmine.db")
	database, err := Open(path)
	if err != nil {
		t.Fatalf("open real embedded schema (requires CGO and -tags fts5): %v", err)
	}
	t.Cleanup(func() {
		if err := database.Close(); err != nil {
			t.Errorf("close database: %v", err)
		}
	})
	var version int
	if err := database.QueryRow("SELECT MAX(version) FROM schema_version").Scan(&version); err != nil {
		t.Fatal(err)
	}
	if version != SchemaVersion {
		t.Fatalf("schema version = %d, want %d", version, SchemaVersion)
	}
	if _, err := database.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(`INSERT INTO users (id, source_type, source_id) VALUES ('user_github_test', 'github', 'test')`); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(`INSERT INTO channels (id, source_type, source_id, name) VALUES ('repo_github_test', 'github', 'test', 'test')`); err != nil {
		t.Fatal(err)
	}
	message := &Message{
		ID: "msg_github_test", SourceType: "github", SourceID: "test",
		Timestamp: time.Now().UTC(), AuthorID: "user_github_test",
		ChannelID: "repo_github_test", Content: "kubernetes deployment troubleshooting",
		NormalizedAt: time.Now().UTC(), SchemaVersion: "2.0",
	}
	if err := database.SaveMessage(message); err != nil {
		t.Fatal(err)
	}
	assertSearch := func(query string, want ...string) {
		t.Helper()
		messages, err := database.SelectMessages(SelectMessagesOptions{SearchText: &query})
		if err != nil {
			t.Fatalf("search %q: %v", query, err)
		}
		var got []string
		for _, message := range messages {
			got = append(got, message.ID)
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("search %q = %v, want %v", query, got, want)
		}
		// Query the index directly too: a stale deleted entry can be hidden by the join.
		var count int
		if err := database.QueryRow("SELECT COUNT(*) FROM messages_fts WHERE messages_fts MATCH ?", query).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != len(want) {
			t.Errorf("index count for %q = %d, want %d", query, count, len(want))
		}
	}
	assertSearch("kubernetes", message.ID)
	assertSearch(`"kubernetes deployment"`, message.ID)
	assertSearch("kubernetes AND deploy*", message.ID)
	assertSearch("unrelated")

	message.Content = "sqlite indexing repaired"
	if err := database.SaveMessage(message); err != nil {
		t.Fatal(err)
	}
	assertSearch("kubernetes")
	assertSearch("sqlite", message.ID)
	if _, err := database.Exec("DELETE FROM messages WHERE id = ?", message.ID); err != nil {
		t.Fatal(err)
	}
	assertSearch("sqlite")

	if err := database.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(path)
	if err != nil {
		t.Fatalf("reopen initialized schema: %v", err)
	}
	if err := reopened.Close(); err != nil {
		t.Fatal(err)
	}
}
