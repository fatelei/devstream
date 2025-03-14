package list

import (
	"fmt"
	"sort"
	"strings"
)

// list is the version of DevStream.
// Assign the value when building with the -X parameter. Example:
// -X github.com/devstream-io/devstream/cmd/devstream/list.PluginsName=${PLUGINS_NAME}
// See the Makefile for more info.

var PluginsName string

// List all of plugins name
func List() {
	listPluginsName := strings.Fields(PluginsName)
	sort.Strings(listPluginsName)
	for _, pluginName := range listPluginsName {
		fmt.Println(pluginName)
	}
}
