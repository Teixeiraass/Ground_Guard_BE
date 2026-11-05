// Package webassets embeds the static HTML pages served directly by the
// backend for links clicked from email (password reset, email verification).
// The app is mobile-only (no web build), so these pages exist purely as the
// browser landing point for those email links.
package webassets

import _ "embed"

//go:embed reset_password.html
var ResetPasswordPage string

//go:embed verify_email.html
var VerifyEmailPage string
