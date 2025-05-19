package common

import (
	"reflect"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

var LogLevels = []string{"trace", "debug", "info", "warn", "warning", "error", "fatal", "panic"}

// AutocompleteLogLevel - Autocomplete log level options.
// -> "trace", "debug", "info", "warn", "error", "fatal", "panic"
func AutocompleteLogLevel(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return LogLevels, cobra.ShellCompDirectiveNoFileComp
}

type formatSuggestion struct {
	fieldname string
	suffix    string
}

func convertFormatSuggestions(suggestions []formatSuggestion) []string {
	completions := make([]string, 0, len(suggestions))
	for _, f := range suggestions {
		completions = append(completions, f.fieldname+f.suffix)
	}
	return completions
}

// AutocompleteFormat - Autocomplete json or a given struct to use for a go template.
// The input can be nil, In this case only json will be autocompleted.
// This function will only work for pointer to structs other types are not supported.
// When "{{." is typed the field and method names of the given struct will be completed.
// This also works recursive for nested structs.
func AutocompleteFormat(o interface{}) func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	// this function provides shell completion for go templates
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		// autocomplete json when nothing or json is typed
		// gocritic complains about the argument order but this is correct in this case
		//nolint:gocritic
		if strings.HasPrefix("json", toComplete) {
			return []string{"json"}, cobra.ShellCompDirectiveNoFileComp
		}

		// no input struct we cannot provide completion return nothing
		if o == nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		// toComplete could look like this: {{ .Config }} {{ .Field.F
		// 1. split the template variable delimiter
		vars := strings.Split(toComplete, "{{")
		if len(vars) == 1 {
			// no variables return no completion
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		// clean the spaces from the last var
		field := strings.Split(vars[len(vars)-1], " ")
		// split this into it struct field names
		fields := strings.Split(field[len(field)-1], ".")
		f := reflect.ValueOf(o)
		if f.Kind() != reflect.Ptr {
			// We panic here to make sure that all callers pass the value by reference.
			// If someone passes a by value then all podman commands will panic since
			// this function is run at init time.
			panic("AutocompleteFormat: passed value must be a pointer to a struct")
		}
		for i := 1; i < len(fields); i++ {
			// last field get all names to suggest
			if i == len(fields)-1 {
				suggestions := getStructFields(f, fields[i])
				// add the current toComplete value in front so that the shell can complete this correctly
				toCompArr := strings.Split(toComplete, ".")
				toCompArr[len(toCompArr)-1] = ""
				toComplete = strings.Join(toCompArr, ".")
				return prefixSlice(toComplete, convertFormatSuggestions(suggestions)), cobra.ShellCompDirectiveNoSpace | cobra.ShellCompDirectiveNoFileComp
			}

			// first follow pointer and create element when it is nil
			f = actualReflectValue(f)
			switch f.Kind() {
			case reflect.Struct:
				for j := 0; j < f.NumField(); j++ {
					field := f.Type().Field(j)
					// ok this is a bit weird but when we have an embedded nil struct
					// calling FieldByName on a name which is present on this struct will panic
					// Therefore we have to init them (non nil ptr), https://github.com/containers/podman/issues/14223
					if field.Anonymous && f.Field(j).Type().Kind() == reflect.Ptr {
						f.Field(j).Set(reflect.New(f.Field(j).Type().Elem()))
					}
				}
				// set the next struct field
				f = f.FieldByName(fields[i])
			case reflect.Map:
				rtype := f.Type().Elem()
				if rtype.Kind() == reflect.Ptr {
					rtype = rtype.Elem()
				}
				f = reflect.New(rtype)
			case reflect.Func:
				if f.Type().NumOut() != 1 {
					// unsupported type return nothing
					return nil, cobra.ShellCompDirectiveNoFileComp
				}
				f = reflect.New(f.Type().Out(0))
			default:
				// unsupported type return nothing
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
		}
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
}

// actualReflectValue takes the value,
// if it is pointer it will dereference it and when it is nil,
// it will create a new value from it
func actualReflectValue(f reflect.Value) reflect.Value {
	// follow the pointer first
	if f.Kind() == reflect.Ptr {
		// if the pointer is nil we create a new value from the elements type
		// this allows us to follow nil pointers and get the actual type
		if f.IsNil() {
			f = reflect.New(f.Type().Elem())
		}
		f = f.Elem()
	}
	return f
}

// getStructFields reads all struct field names and method names and returns them.
func getStructFields(f reflect.Value, prefix string) []formatSuggestion {
	var suggestions []formatSuggestion
	if f.IsValid() {
		suggestions = append(suggestions, getMethodNames(f, prefix)...)
	}

	f = actualReflectValue(f)
	// we only support structs
	if f.Kind() != reflect.Struct {
		return suggestions
	}

	var anonymous []formatSuggestion
	// loop over all field names
	for j := 0; j < f.NumField(); j++ {
		field := f.Type().Field(j)
		// check if struct field is not exported, templates only use exported fields
		// PkgPath is always empty for exported fields
		if field.PkgPath != "" {
			continue
		}
		fname := field.Name
		suffix := "}}"
		kind := field.Type.Kind()
		if kind == reflect.Ptr {
			// make sure to read the actual type when it is a pointer
			kind = field.Type.Elem().Kind()
		}
		// when we have a nested struct do not append braces instead append a dot
		if kind == reflect.Struct || kind == reflect.Map {
			suffix = "."
		}
		// if field is anonymous add the child fields as well
		if field.Anonymous {
			anonymous = append(anonymous, getStructFields(f.Field(j), prefix)...)
		}
		if strings.HasPrefix(fname, prefix) {
			// add field name with suffix
			suggestions = append(suggestions, formatSuggestion{fieldname: fname, suffix: suffix})
		}
	}
outer:
	for _, ano := range anonymous {
		// we should only add anonymous child fields if they are not already present.
		for _, sug := range suggestions {
			if ano.fieldname == sug.fieldname {
				continue outer
			}
		}
		suggestions = append(suggestions, ano)
	}
	return suggestions
}

func getMethodNames(f reflect.Value, prefix string) []formatSuggestion {
	suggestions := make([]formatSuggestion, 0, f.NumMethod())
	for j := 0; j < f.NumMethod(); j++ {
		method := f.Type().Method(j)
		// in a template we can only run functions with one return value
		if method.Func.Type().NumOut() != 1 {
			continue
		}
		// when we have a nested struct do not append braces instead append a dot
		kind := method.Func.Type().Out(0).Kind()
		suffix := "}}"
		if kind == reflect.Struct || kind == reflect.Map {
			suffix = "."
		}
		// From a template user's POV it is not important whether they use a struct field or method.
		// They only notice the difference when the function requires arguments.
		// So let's be nice and let the user know that this method requires arguments via the help text.
		// Note since this is actually a method on a type the first argument is always fix so we should skip it.
		num := method.Func.Type().NumIn() - 1
		if num > 0 {
			// everything after tab will the completion scripts show as help when enabled
			// overwrite the suffix because it expects the args
			suffix = "\tThis is a function and requires " + strconv.Itoa(num) + " argument"
			if num > 1 {
				// add plural s
				suffix += "s"
			}
		}
		fname := method.Name
		if strings.HasPrefix(fname, prefix) {
			// add method name with closing braces
			suggestions = append(suggestions, formatSuggestion{fieldname: fname, suffix: suffix})
		}
	}
	return suggestions
}

func prefixSlice(pre string, slice []string) []string {
	for i := range slice {
		slice[i] = pre + slice[i]
	}
	return slice
}
