package getconf

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"
)

func parseTags(t reflect.StructField, o *Option, prefix string) error {
	o.name = strings.ToLower(prefix + t.Name)
	o.oType = t.Type.Kind()
	if tag, exists := t.Tag.Lookup("getconf"); exists {
		if tag = strings.TrimSpace(tag); tag != "" {
			if tag == "-" {
				return errors.New("untrack")
			}

			name, opts := parseTag(tag)
			if name != "" {
				o.name = strings.ToLower(prefix + name)
			}
			k := strings.Split(opts, ",")

			for _, sk := range k {
				if sk == "" {
					continue
				}
				key, value := getKeyValFromTagOption(sk)
				switch key {
				case "default":
					o.defValue = value
					o.value = getTypedValue(o.defValue, o.oType)
					o.updatedAt = time.Now().UTC()
					o.lastSetBy = "default"
				case "info":
					o.usage = value
				}
			}
		}
	}
	return nil
}

func getKeyValFromTagOption(opt string) (string, string) {
	if idx := strings.Index(opt, ":"); idx != -1 {
		return strings.TrimSpace(opt[:idx]), strings.TrimSpace(opt[idx+1:])

	}
	fmt.Printf("ilegal option: %s\n\n", opt)
	return "", ""
}

func parseTag(tag string) (string, string) {
	if idx := strings.Index(tag, ","); idx != -1 {
		return tag[:idx], strings.TrimSpace(tag[idx+1:])
	}
	return tag, ""
}
