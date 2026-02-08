# Identity Resolution Design

## Current State (v2.0)

User and channel names are resolved **within each source**:
- Slack user ID `U45678` → display name from Slack workspace
- GitHub user ID `12345` → username from GitHub
- Channel IDs → channel names from their respective sources

Table output shows human-readable names, but they're source-specific.

## Future: Cross-Platform Identity Resolution

### Goal
Link identities across platforms so that:
- GitHub `@solvaholic` = Slack `U45678` = canonical identity `solvaholic`
- All references resolve to the same display name in output

### Approach

#### 1. Automatic Matching (Email-based)
```sql
-- Find users with matching emails across sources
SELECT u1.id as github_id, u2.id as slack_id, u1.email
FROM users u1
JOIN users u2 ON u1.email = u2.email AND u1.source_type != u2.source_type
WHERE u1.email IS NOT NULL AND u1.email != '';
```

Create canonical identities for matches:
```go
identity := &db.Identity{
    CanonicalID:   "identity_solvaholic",
    CanonicalName: "solvaholic",
    PrimaryEmail:  "user@example.com",
    Confidence:    0.95, // High confidence for email match
}
```

Link users to canonical identity:
```go
db.LinkUserToIdentity("user_github_12345", "identity_solvaholic")
db.LinkUserToIdentity("user_slack_U45678", "identity_solvaholic")
```

#### 2. Manual Mapping
For users without matching emails, provide a command:
```bash
mine identity link --github @solvaholic --slack @solvaholic
mine identity link --email user@example.com --slack U45678
```

#### 3. Display Name Resolution
When outputting results:
1. Check if user has `canonical_id`
2. Look up identity by `canonical_id`
3. Use identity's `canonical_name` for display
4. Fall back to source-specific name if no identity

```go
func resolveDisplayName(userID string) string {
    user := db.GetUser(userID)
    if user.CanonicalID != nil {
        identity := db.GetIdentity(*user.CanonicalID)
        if identity != nil {
            return identity.CanonicalName
        }
    }
    // Fall back to source name
    return user.DisplayName or user.RealName
}
```

### Implementation Steps (Future)

1. **Phase 1**: Email-based matching
   - Implement `ResolveIdentities()` function
   - Create identities for email matches
   - Link users automatically

2. **Phase 2**: Manual linking
   - Add `mine identity link` command
   - Add `mine identity list` to show mappings
   - Add `mine identity unlink` to break links

3. **Phase 3**: Name resolution
   - Update table output to use canonical names
   - Add flag `--use-identity` to enable/disable
   - Show source badge (📧 for email, 💬 for Slack, etc.)

4. **Phase 4**: Confidence scoring
   - Track confidence for each identity link
   - Show warnings for low-confidence matches
   - Allow user to confirm/reject matches

### Database Schema (Already Present)

The schema already supports this:
```sql
-- identities table: canonical identities
-- users.canonical_id: links user to identity
```

### Example Output (Future)

```
TIMESTAMP           AUTHOR              CHANNEL             CONTENT
2026-02-07 10:30   solvaholic 💬       #engineering       How do I...
2026-02-07 10:35   solvaholic 📧       threadmine#42      Try this...
```

Where:
- `solvaholic` is the canonical name
- 💬 indicates this is from Slack
- 📧 indicates this is from GitHub
- Both resolve to the same canonical identity

## Notes

- Don't force identity resolution - some users may want source-specific views
- Confidence scores are important to avoid false matches
- Consider privacy: don't automatically link identities without user consent
- Email matching should be opt-in with `--resolve-identities` flag
