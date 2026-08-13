-- Allow the opencode platform in user_platform_quotas.platform CHECK.
--
-- Background: the OpenCode Go platform was added to groups and accounts, but
-- never to the per-user platform quota subsystem. PlatformOpencode existed as
-- a constant while AllowedQuotaPlatforms, the ent build-time validator and
-- this CHECK all still listed only anthropic/openai/gemini/antigravity/grok,
-- so admins could not set a daily/weekly/monthly limit for opencode at all --
-- the option never rendered, and a direct API call was rejected by the DB.
--
-- Keep in sync with internal/service/domain_constants.go AllowedQuotaPlatforms
-- and ent/schema/user_platform_quota.go. DROP ... IF EXISTS keeps this
-- re-runnable; the new constraint is a superset of the old one, so existing
-- rows validate instantly.
ALTER TABLE user_platform_quotas
    DROP CONSTRAINT IF EXISTS user_platform_quotas_platform_check;

ALTER TABLE user_platform_quotas
    ADD CONSTRAINT user_platform_quotas_platform_check
    CHECK (platform IN ('anthropic', 'openai', 'gemini', 'antigravity', 'grok', 'opencode'));
