package config

import (
	"os"
	"strconv"
)

type Config struct {
	HTTPAddr, ProfilerDatabaseURL, RedisURL, LogLevel string
	JobWorkers                                        int
}

func Load() Config {
	return Config{HTTPAddr: get("HTTP_ADDR", ":8080"), ProfilerDatabaseURL: get("PROFILER_DATABASE_URL", ""), RedisURL: get("REDIS_URL", ""), LogLevel: get("LOG_LEVEL", "info"), JobWorkers: getInt("JOB_WORKERS", 4)}
}
func get(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
func getInt(k string, d int) int {
	v, e := strconv.Atoi(get(k, ""))
	if e != nil || v < 1 {
		return d
	}
	return v
}
