package config

// CompiledSecret holds the embedded GOTRAY_SECRET provided at build time via
// -ldflags. When empty, the application will fall back to reading the
// GOTRAY_SECRET environment variable for local development.
var CompiledSecret string
