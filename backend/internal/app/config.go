package app

import "log/slog"

type Config struct {
	LogLevel                 slog.Level `envconfig:"LOG_LEVEL"`
	Port                     int        `envconfig:"PORT"                        required:"true"`
	ExternalWebServerAddress string     `envconfig:"EXTERNAL_WEB_SERVER_ADDRESS"`
	DatabasePath             string     `envconfig:"DATABASE_PATH"               required:"true"`
	WebAuthnRPID             string     `envconfig:"WEBAUTHN_RP_ID"              required:"true"`
	WebAuthnOrigin           string     `envconfig:"WEBAUTHN_ORIGIN"             required:"true"`
}
