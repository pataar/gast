// Package demo provides fake event data for screenshots and demos.
package demo

import (
	"time"

	"github.com/pataar/gast/internal/event"
)

// Events returns a set of realistic-looking fake events for demo/screenshot
// purposes. Events are returned newest-first (like the API) so the TUI's
// mergeEvents will reverse them into chronological order.
func Events() []event.Event {
	now := time.Now()
	m := func(mins int) time.Time { return now.Add(-time.Duration(mins) * time.Minute) }

	events := []event.Event{
		// --- oldest ---
		{
			ID: 1, ActionName: "opened", AuthorUsername: "lena.berg",
			CreatedAt: m(240), ProjectName: "acme/infrastructure/k8s-configs",
			TargetType: "Issue", TargetIID: 78, TargetTitle: "Ingress TLS certificate auto-renewal failing on staging",
		},
		{
			ID: 2, AuthorUsername: "james.chen",
			CreatedAt: m(235), ProjectName: "acme/backend/notification-service",
			PushData: &event.PushData{CommitCount: 4, Ref: "feat/email-templates", CommitTitle: "feat: add mjml email template engine"},
		},
		{
			ID: 3, ActionName: "commented on", AuthorUsername: "sophie.martin",
			CreatedAt: m(230), ProjectName: "acme/infrastructure/k8s-configs",
			TargetType: "Issue", TargetIID: 78, TargetTitle: "Ingress TLS certificate auto-renewal failing on staging",
			NoteBody: "I ran into this last week too. The cert-manager ClusterIssuer needs updating after the Helm upgrade.",
		},
		{
			ID: 4, ActionName: "opened", AuthorUsername: "sophie.martin",
			CreatedAt: m(225), ProjectName: "acme/backend/notification-service",
			TargetType: "MergeRequest", TargetIID: 332, TargetTitle: "Refactor email queue to use Redis Streams",
		},
		{
			ID: 5, AuthorUsername: "sophie.martin",
			CreatedAt: m(220), ProjectName: "acme/backend/notification-service",
			PushData: &event.PushData{CommitCount: 2, Ref: "refactor/redis-streams", CommitTitle: "refactor: migrate from bull to redis streams"},
		},
		{
			ID: 6, ActionName: "commented on", AuthorUsername: "alex.rivera",
			CreatedAt: m(215), ProjectName: "acme/backend/notification-service",
			TargetType: "MergeRequest", TargetIID: 332, TargetTitle: "Refactor email queue to use Redis Streams",
			NoteBody: "Nice approach. One concern: are we handling consumer group recovery for crashed workers?",
		},
		{
			ID: 7, ActionName: "opened", AuthorUsername: "maria.kovacs",
			CreatedAt: m(210), ProjectName: "acme/frontend/dashboard",
			TargetType: "Issue", TargetIID: 201, TargetTitle: "Table sorting breaks when column contains null values",
		},
		{
			ID: 8, AuthorUsername: "pieter.willekens",
			CreatedAt: m(205), ProjectName: "acme/payments/api",
			PushData: &event.PushData{CommitCount: 1, Ref: "fix/idempotency-key", CommitTitle: "fix: deduplicate webhook retries using idempotency key"},
		},
		{
			ID: 9, ActionName: "approved", AuthorUsername: "james.chen",
			CreatedAt: m(200), ProjectName: "acme/backend/notification-service",
			TargetType: "MergeRequest", TargetIID: 332, TargetTitle: "Refactor email queue to use Redis Streams",
		},
		{
			ID: 10, ActionName: "accepted", AuthorUsername: "sophie.martin",
			CreatedAt: m(198), ProjectName: "acme/backend/notification-service",
			TargetType: "MergeRequest", TargetIID: 332, TargetTitle: "Refactor email queue to use Redis Streams",
		},
		{
			ID: 11, ActionName: "commented on", AuthorUsername: "group_482_bot",
			CreatedAt: m(195), ProjectName: "acme/payments/api",
			TargetType: "DiscussionNote", TargetIID: 0,
			NoteBody: "Pipeline #48291 passed. All 347 tests green. Coverage: 91.2%",
		},
		{
			ID: 12, ActionName: "opened", AuthorUsername: "pieter.willekens",
			CreatedAt: m(190), ProjectName: "acme/payments/api",
			TargetType: "MergeRequest", TargetIID: 894, TargetTitle: "Fix webhook idempotency for Stripe retries",
		},
		{
			ID: 13, AuthorUsername: "lena.berg",
			CreatedAt: m(185), ProjectName: "acme/infrastructure/k8s-configs",
			PushData: &event.PushData{CommitCount: 1, Ref: "fix/cert-manager", CommitTitle: "fix: update ClusterIssuer to use ACME v2 endpoint"},
		},
		{
			ID: 14, ActionName: "commented on", AuthorUsername: "james.chen",
			CreatedAt: m(180), ProjectName: "acme/payments/api",
			TargetType: "MergeRequest", TargetIID: 894, TargetTitle: "Fix webhook idempotency for Stripe retries",
			NoteBody: "Should we also add a TTL on the idempotency cache? Otherwise it could grow unbounded.",
		},
		{
			ID: 15, ActionName: "commented on", AuthorUsername: "pieter.willekens",
			CreatedAt: m(175), ProjectName: "acme/payments/api",
			TargetType: "MergeRequest", TargetIID: 894, TargetTitle: "Fix webhook idempotency for Stripe retries",
			NoteBody: "Good call. Added a 24h TTL, Stripe retries are guaranteed to stop after 72h anyway.",
		},
		{
			ID: 16, AuthorUsername: "alex.rivera",
			CreatedAt: m(170), ProjectName: "acme/frontend/dashboard",
			PushData: &event.PushData{CommitCount: 1, Ref: "fix/table-null-sort", CommitTitle: "fix: handle null values in table column comparator"},
		},
		{
			ID: 17, ActionName: "closed", AuthorUsername: "alex.rivera",
			CreatedAt: m(168), ProjectName: "acme/frontend/dashboard",
			TargetType: "Issue", TargetIID: 201, TargetTitle: "Table sorting breaks when column contains null values",
		},
		{
			ID: 18, ActionName: "approved", AuthorUsername: "sophie.martin",
			CreatedAt: m(165), ProjectName: "acme/payments/api",
			TargetType: "MergeRequest", TargetIID: 894, TargetTitle: "Fix webhook idempotency for Stripe retries",
		},
		{
			ID: 19, ActionName: "accepted", AuthorUsername: "pieter.willekens",
			CreatedAt: m(163), ProjectName: "acme/payments/api",
			TargetType: "MergeRequest", TargetIID: 894, TargetTitle: "Fix webhook idempotency for Stripe retries",
		},
		{
			ID: 20, AuthorUsername: "pieter.willekens",
			CreatedAt: m(160), ProjectName: "acme/payments/api",
			PushData: &event.PushData{CommitCount: 1, Ref: "main", CommitTitle: "chore(release): v2.4.1"},
		},
		{
			ID: 21, AuthorUsername: "pieter.willekens",
			CreatedAt: m(160), ProjectName: "acme/payments/api",
			PushData: &event.PushData{CommitCount: 1, Ref: "v2.4.1", CommitTitle: "chore(release): v2.4.1"},
		},
		{
			ID: 22, ActionName: "closed", AuthorUsername: "lena.berg",
			CreatedAt: m(155), ProjectName: "acme/infrastructure/k8s-configs",
			TargetType: "Issue", TargetIID: 78, TargetTitle: "Ingress TLS certificate auto-renewal failing on staging",
		},
		{
			ID: 23, ActionName: "opened", AuthorUsername: "james.chen",
			CreatedAt: m(150), ProjectName: "acme/backend/user-service",
			TargetType: "MergeRequest", TargetIID: 217, TargetTitle: "Add rate limiting to password reset endpoint",
		},
		{
			ID: 24, AuthorUsername: "james.chen",
			CreatedAt: m(148), ProjectName: "acme/backend/user-service",
			PushData: &event.PushData{CommitCount: 3, Ref: "feat/rate-limit", CommitTitle: "feat: sliding window rate limiter with Redis"},
		},
		{
			ID: 25, ActionName: "commented on", AuthorUsername: "maria.kovacs",
			CreatedAt: m(140), ProjectName: "acme/backend/user-service",
			TargetType: "MergeRequest", TargetIID: 217, TargetTitle: "Add rate limiting to password reset endpoint",
			NoteBody: "Can we make the window size and max attempts configurable via environment variables?",
		},
		{
			ID: 26, AuthorUsername: "alex.rivera",
			CreatedAt: m(135), ProjectName: "acme/frontend/dashboard",
			PushData: &event.PushData{CommitCount: 5, Ref: "feat/chart-migration", CommitTitle: "feat: migrate bar charts from Chart.js to D3"},
		},
		{
			ID: 27, ActionName: "opened", AuthorUsername: "alex.rivera",
			CreatedAt: m(130), ProjectName: "acme/frontend/dashboard",
			TargetType: "MergeRequest", TargetIID: 562, TargetTitle: "Migrate chart components to D3.js",
		},
		{
			ID: 28, ActionName: "commented on", AuthorUsername: "sophie.martin",
			CreatedAt: m(125), ProjectName: "acme/frontend/dashboard",
			TargetType: "MergeRequest", TargetIID: 562, TargetTitle: "Migrate chart components to D3.js",
			NoteBody: "The bundle size diff looks great. From 148kb to 62kb for the charts module.",
		},
		{
			ID: 29, ActionName: "opened", AuthorUsername: "lena.berg",
			CreatedAt: m(120), ProjectName: "acme/infrastructure/terraform",
			TargetType: "Issue", TargetIID: 234, TargetTitle: "Upgrade PostgreSQL from 15 to 16",
		},
		{
			ID: 30, ActionName: "commented on", AuthorUsername: "project_1094_bot_a7e3f91bc2d445e8901234567890abcd",
			CreatedAt: m(115), ProjectName: "acme/frontend/dashboard",
			TargetType: "DiscussionNote", TargetIID: 562,
			NoteBody: "Lighthouse report: Performance 94 (+7), Accessibility 100, Best Practices 100.",
		},
		{
			ID: 31, AuthorUsername: "maria.kovacs",
			CreatedAt: m(110), ProjectName: "acme/backend/auth-service",
			PushData: &event.PushData{CommitCount: 2, Ref: "feat/oidc-discovery", CommitTitle: "feat: auto-discover OIDC provider configuration"},
		},
		{
			ID: 32, ActionName: "opened", AuthorUsername: "maria.kovacs",
			CreatedAt: m(108), ProjectName: "acme/backend/auth-service",
			TargetType: "MergeRequest", TargetIID: 1447, TargetTitle: "Implement OIDC provider discovery",
		},
		{
			ID: 33, ActionName: "commented on", AuthorUsername: "pieter.willekens",
			CreatedAt: m(100), ProjectName: "acme/backend/auth-service",
			TargetType: "MergeRequest", TargetIID: 1447, TargetTitle: "Implement OIDC provider discovery",
			NoteBody: "The discovery endpoint caching looks solid. Have you tested with providers that rotate their JWKS frequently?",
		},
		{
			ID: 34, ActionName: "approved", AuthorUsername: "alex.rivera",
			CreatedAt: m(95), ProjectName: "acme/frontend/dashboard",
			TargetType: "MergeRequest", TargetIID: 562, TargetTitle: "Migrate chart components to D3.js",
		},
		{
			ID: 35, ActionName: "commented on", AuthorUsername: "james.chen",
			CreatedAt: m(90), ProjectName: "acme/backend/auth-service",
			TargetType: "MergeRequest", TargetIID: 1447, TargetTitle: "Implement OIDC provider discovery",
			NoteBody: "Nit: the fallback timeout of 30s seems high. Most OIDC providers respond in <500ms.",
		},
		{
			ID: 36, AuthorUsername: "maria.kovacs",
			CreatedAt: m(85), ProjectName: "acme/backend/auth-service",
			PushData: &event.PushData{CommitCount: 1, Ref: "feat/oidc-discovery", CommitTitle: "fix: reduce OIDC discovery timeout to 5s"},
		},
		{
			ID: 37, ActionName: "accepted", AuthorUsername: "alex.rivera",
			CreatedAt: m(80), ProjectName: "acme/frontend/dashboard",
			TargetType: "MergeRequest", TargetIID: 562, TargetTitle: "Migrate chart components to D3.js",
		},
		{
			ID: 38, ActionName: "commented on", AuthorUsername: "group_482_bot",
			CreatedAt: m(78), ProjectName: "acme/backend/auth-service",
			TargetType: "DiscussionNote", TargetIID: 1447,
			NoteBody: "Security scan complete. No vulnerabilities found. SAST/DAST passed.",
		},
		{
			ID: 39, ActionName: "approved", AuthorUsername: "pieter.willekens",
			CreatedAt: m(75), ProjectName: "acme/backend/auth-service",
			TargetType: "MergeRequest", TargetIID: 1447, TargetTitle: "Implement OIDC provider discovery",
		},
		{
			ID: 40, ActionName: "accepted", AuthorUsername: "maria.kovacs",
			CreatedAt: m(73), ProjectName: "acme/backend/auth-service",
			TargetType: "MergeRequest", TargetIID: 1447, TargetTitle: "Implement OIDC provider discovery",
		},
		{
			ID: 41, AuthorUsername: "lena.berg",
			CreatedAt: m(70), ProjectName: "acme/infrastructure/terraform",
			PushData: &event.PushData{CommitCount: 1, Ref: "feat/pg16-upgrade", CommitTitle: "feat: add PostgreSQL 16 module with blue-green setup"},
		},
		{
			ID: 42, ActionName: "commented on", AuthorUsername: "alex.rivera",
			CreatedAt: m(65), ProjectName: "acme/infrastructure/terraform",
			TargetType: "Issue", TargetIID: 234, TargetTitle: "Upgrade PostgreSQL from 15 to 16",
			NoteBody: "We should schedule this during the next maintenance window. @pieter.willekens can you coordinate with the DBA team?",
		},
		{
			ID: 43, AuthorUsername: "sophie.martin",
			CreatedAt: m(60), ProjectName: "acme/backend/notification-service",
			PushData: &event.PushData{CommitCount: 1, Ref: "main", CommitTitle: "chore: bump redis client to v5.2.0"},
		},
		{
			ID: 44, ActionName: "opened", AuthorUsername: "james.chen",
			CreatedAt: m(55), ProjectName: "acme/backend/user-service",
			TargetType: "Issue", TargetIID: 89, TargetTitle: "Session tokens not invalidated after password change",
		},
		{
			ID: 45, ActionName: "approved", AuthorUsername: "lena.berg",
			CreatedAt: m(50), ProjectName: "acme/backend/user-service",
			TargetType: "MergeRequest", TargetIID: 217, TargetTitle: "Add rate limiting to password reset endpoint",
		},
		{
			ID: 46, ActionName: "accepted", AuthorUsername: "james.chen",
			CreatedAt: m(48), ProjectName: "acme/backend/user-service",
			TargetType: "MergeRequest", TargetIID: 217, TargetTitle: "Add rate limiting to password reset endpoint",
		},
		{
			ID: 47, AuthorUsername: "alex.rivera",
			CreatedAt: m(42), ProjectName: "acme/frontend/dashboard",
			PushData: &event.PushData{CommitCount: 1, Ref: "main", CommitTitle: "fix: correct date picker timezone offset"},
		},
		{
			ID: 48, ActionName: "commented on", AuthorUsername: "maria.kovacs",
			CreatedAt: m(35), ProjectName: "acme/frontend/dashboard",
			TargetType: "MergeRequest", TargetIID: 562, TargetTitle: "Migrate chart components to D3.js",
			NoteBody: "The tooltip positioning on the new line charts is slightly off on Safari. Can you check?",
		},
		{
			ID: 49, AuthorUsername: "pieter.willekens",
			CreatedAt: m(28), ProjectName: "acme/infrastructure/terraform",
			PushData: &event.PushData{CommitCount: 1, Ref: "main", CommitTitle: "docs: add PostgreSQL 16 migration runbook"},
		},
		{
			ID: 50, ActionName: "commented on", AuthorUsername: "sophie.martin",
			CreatedAt: m(20), ProjectName: "acme/backend/user-service",
			TargetType: "Issue", TargetIID: 89, TargetTitle: "Session tokens not invalidated after password change",
			NoteBody: "This is a security concern. We should add token version tracking to the session store. @pieter.willekens thoughts?",
		},
	}

	// Reverse to newest-first order, matching the real GitLab API response.
	for i, j := 0, len(events)-1; i < j; i, j = i+1, j-1 {
		events[i], events[j] = events[j], events[i]
	}

	return events
}
