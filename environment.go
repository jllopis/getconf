package getconf

import (
	"os"
	"strings"
)

func loadFromEnv(gc *GetConf, opts *GetConfOptions) {
	for _, o := range gc.options {
		val := getEnv(opts.EnvPrefix, o.name)
		if val != "" {
			gc.setOption(o.name, val, "env")
		}
	}
}

// getEnv Looks up variable in environment.
// env variables must be uppercase and the only separator allowed in the underscore. Dots and middle score
// will be changed to underscore.
// The variable name must be preceded by a prefix. If no prefix is provided the variable will be ignored.
// Taken from https://github.com/rakyll/globalconf/blob/master/globalconf.go#L159
func getEnv(envPrefix, flagName string) string {
	// If we haven't set an EnvPrefix, don't lookup vals in the ENV
	if envPrefix == "" {
		return ""
	}
	if !strings.HasSuffix(envPrefix, "_") {
		envPrefix += "_"
	}
	flagName = strings.Replace(flagName, ".", "_", -1)
	flagName = strings.Replace(flagName, "-", "_", -1)
	envKey := strings.ToUpper(envPrefix + flagName)
	return os.Getenv(envKey)
}
