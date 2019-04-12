package getconf

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"
)

// parseTags detects getconf tags in the t reflect.StructField. The pointer
// to Option param is passed to allow settings that can be provided in the flag and
// the prefix is used to build nested variables and represents its parents.
//
//
//    * name of the variable to be used when calling it. Is should be provided as the first
//      element in the tag
//    * default: default value of the variable
//    * info: document the purpose of the variable
//    * - : a dash should be the only element in the tag. Discards the variable
//
// getconf tags are comma separated so no comma is allowed in the options. If a name is not
// indicated, the comma must appear before any other option: ", default: ....."
//
// After each identifier must follow a colon (:) and after it the value.
//
// Ex: MyOption   string  `my-opt-name, default: goodOption, info: a test option`
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

// getKeyValFromTagOption returns the key, value pair that can be extracted
// from opt.
//
// opt is a string with key and value separated by a colon: "key: value" as
// appears in getconf tags.
func getKeyValFromTagOption(opt string) (string, string) {
	if idx := strings.Index(opt, ":"); idx != -1 {
		return strings.TrimSpace(opt[:idx]), strings.TrimSpace(opt[idx+1:])

	}
	fmt.Printf("ilegal option: %s\n\n", opt)
	return "", ""
}

// parseTag will parse the tag string and returns two strings:
//
// - the first one corresponding to the name of the variable. If no name is
//    provided by the tag, ths struct name is used
// - the string containing the options, comma separated
func parseTag(tag string) (string, string) {
	if idx := strings.Index(tag, ","); idx != -1 {
		return tag[:idx], strings.TrimSpace(tag[idx+1:])
	}
	return tag, ""
}
