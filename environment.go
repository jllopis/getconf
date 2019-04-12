package getconf

import (
	"os"
	"strings"
)

func loadFromEnv() {
	for _, o := range g2.options {
		val := getEnv(g2.envPrefix, o.name, g2.keyDelim)
		if val != "" {
			g2.setOption(o.name, val, "env")
		}
	}
}

// getEnv Looks up variable in environment.
// env variables must be uppercase and the only separator allowed in the underscore. Dots and middle score
// will be changed to underscore.
//
// The variable name must be preceded by a prefix. If no prefix is provided the variable will be ignored.
//
// There can be nested variables by using 'separator'. By defalut, separator is '__'. They will be transformed
// from '::' when the key is set for reading from ENV:
//
//     ex: parent::child -> GC2_PARENT__CHILD
//
// Taken from https://github.com/rakyll/globalconf/blob/master/globalconf.go#L159
func getEnv(envPrefix, flagName, keyDelim string) string {
	// If we haven't set an EnvPrefix, don't lookup vals in the ENV
	if envPrefix == "" {
		return ""
	}
	if !strings.HasSuffix(envPrefix, "_") {
		envPrefix += "_"
	}
	flagName = strings.Replace(flagName, "-", "_", -1)
	flagName = strings.Replace(flagName, keyDelim, "__", -1)
	flagName = strings.Replace(flagName, ".", "_", -1)
	envKey := strings.ToUpper(envPrefix + flagName)

	return os.Getenv(envKey)
}
