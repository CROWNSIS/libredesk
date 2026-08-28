package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V2_8_1_Oledu_1 adds foreign-keyed per-assistant knowledge and inbox scopes.
func V2_8_1_Oledu_1(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS ai_assistant_help_centers (
			id SERIAL PRIMARY KEY,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			assistant_id INTEGER NOT NULL REFERENCES ai_assistants(id) ON DELETE CASCADE,
			help_center_id INTEGER NOT NULL REFERENCES help_centers(id) ON DELETE CASCADE,
			CONSTRAINT unique_ai_assistant_help_center UNIQUE (assistant_id, help_center_id)
		);
		CREATE INDEX IF NOT EXISTS index_ai_assistant_help_centers_on_assistant_id ON ai_assistant_help_centers(assistant_id);
		CREATE INDEX IF NOT EXISTS index_ai_assistant_help_centers_on_help_center_id ON ai_assistant_help_centers(help_center_id);

		CREATE TABLE IF NOT EXISTS ai_assistant_inboxes (
			id SERIAL PRIMARY KEY,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			assistant_id INTEGER NOT NULL REFERENCES ai_assistants(id) ON DELETE CASCADE,
			inbox_id INTEGER NOT NULL REFERENCES inboxes(id) ON DELETE CASCADE,
			CONSTRAINT unique_ai_assistant_inbox UNIQUE (assistant_id, inbox_id)
		);
		CREATE INDEX IF NOT EXISTS index_ai_assistant_inboxes_on_assistant_id ON ai_assistant_inboxes(assistant_id);
		CREATE INDEX IF NOT EXISTS index_ai_assistant_inboxes_on_inbox_id ON ai_assistant_inboxes(inbox_id);

		-- This column existed only in an unreleased fork build. Copy it forward if present,
		-- then remove the unenforceable array representation.
		DO $$
		BEGIN
			IF EXISTS (
				SELECT 1 FROM information_schema.columns
				WHERE table_name = 'ai_assistants' AND column_name = 'help_center_ids'
			) THEN
				INSERT INTO ai_assistant_help_centers (assistant_id, help_center_id)
				SELECT a.id, unnest(a.help_center_ids) FROM ai_assistants a
				ON CONFLICT (assistant_id, help_center_id) DO NOTHING;
				ALTER TABLE ai_assistants DROP COLUMN help_center_ids;
			END IF;
		END $$;
	`)
	return err
}
