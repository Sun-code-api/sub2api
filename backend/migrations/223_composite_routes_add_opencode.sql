-- Allow opencode as a composite route target.
--
-- The frontend types composite target_platform as
-- Exclude<GroupPlatform, 'composite'>, which includes opencode, so the UI
-- offered it while the handler binding (oneof=...) and this CHECK rejected
-- it -- picking OpenCode in a composite group failed with a 400. Widen the
-- constraint so composite groups can route models to the opencode pool.
--
-- Keep in sync with the TargetPlatform binding in
-- internal/handler/admin/group_handler.go.
ALTER TABLE composite_model_routes
    DROP CONSTRAINT IF EXISTS composite_model_routes_target_platform_check;

ALTER TABLE composite_model_routes
    ADD CONSTRAINT composite_model_routes_target_platform_check
    CHECK (target_platform IN ('anthropic', 'openai', 'gemini', 'antigravity', 'grok', 'opencode'));
