
ALTER TABLE team_remote_member_view_tokens ADD COLUMN ptk_role_type SMALLINT NOT NULL DEFAULT 2; -- 2 == "admin"
ALTER TABLE team_remote_member_view_tokens ADD COLUMN ptk_viz_level SMALLINT NOT NULL DEFAULT 0;