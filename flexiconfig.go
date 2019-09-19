// Package flexiconfig is a configuration package with the goal of being
//  powerful but not be more complex than a configuration package should
//  be.
// FlexiConfig is a hierarchical system that will merge configs
//  together based on the order that they are loaded. Later config loads
//  will replace earlier settings if they overlap.
// A core part of this package is the ability to load lua files. This
//  gives you the ability to run a sub program in order to generate your
//  config.
package flexiconfig

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	lua "github.com/yuin/gopher-lua"

	"github.com/mitchellh/mapstructure"
	luajson "layeh.com/gopher-json"
)

// LuaLoader is the type representing the function signature used to
//  load custom lua modules.
type LuaLoader func(L *lua.LState) int

// Settings is the main type that holds the config and loads new
//  configuration files.
type Settings struct {
	settings   map[string]interface{}
	luaModules map[string]lua.LGFunction
}

// NewSettings creates a new empty settings struct.
func NewSettings() Settings {
	settings := Settings{}
	settings.settings = make(map[string]interface{})
	settings.luaModules = make(map[string]lua.LGFunction)

	return settings
}

// Print is a utility function to print out the settings as JSON
func (this Settings) Print() {
	b := this.GetPrettyJSON("", "  ")
	fmt.Println(string(b))
}

// GetPrettyJSON returns a pretty formatted json of the current config
func (this Settings) GetPrettyJSON(prefix, indent string) []byte {
	b, err := json.MarshalIndent(this.settings, prefix, indent)
	if err != nil {
		panic(err)
	}
	return b
}

// GetJSON returns the json representation of the current config. This is useful
//  to retain a static copy of the settings for later.
func (this Settings) GetJSON() []byte {
	b, err := json.Marshal(this.settings)
	if err != nil {
		panic(err)
	}
	return b
}

// AddLuaLoader can be used to add a custom lua module to each lua-state that is
//  used. Note that flexiconfig currently creates a new lua instance for every
//  lua config file loaded.
func (this *Settings) AddLuaLoader(name string, loader lua.LGFunction) {
	this.luaModules[name] = loader
}

// LoadLuaString is used to load a config file from a lua string.
func (this *Settings) LoadLuaString(code string) error {
	L := lua.NewState()
	luajson.Preload(L)
	defer L.Close()

	for moduleName, loader := range this.luaModules {
		L.PreloadModule(moduleName, loader)
	}
	if err := L.DoString(code); err != nil {
		return err
	}

	return this.loadLuaState(L.Get(-1))
}

// LoadLuaFile is used to load a lua config file from a specified path
func (this *Settings) LoadLuaFile(path string) error {
	L := lua.NewState()
	luajson.Preload(L)
	defer L.Close()

	for moduleName, loader := range this.luaModules {
		L.PreloadModule(moduleName, loader)
	}

	if err := L.DoFile(path); err != nil {
		return err
	}

	return this.loadLuaState(L.Get(-1))
}

// loadLuaState is used to load the lua value into the current settings.
func (this *Settings) loadLuaState(lv lua.LValue) error {
	jsonSettings, err := luajson.Encode(lv)
	if err != nil {
		return err
	}

	return this.LoadJSON(jsonSettings)
}

// LoadJSON takes a byte slice, dejsonifys it, then stores the contents in the
//  Settings object.
func (this *Settings) LoadJSON(b []byte) error {
	var newSettings map[string]interface{}
	err := json.Unmarshal(b, &newSettings)

	if err != nil {
		return err
	}

	return this.MergeSettings(newSettings)
}

// LoadJSON takes a path to a .json file and loads it into the Settings object.
func (this *Settings) LoadJSONFile(path string) error {
	// Just a bit of silly. No more than a bit
	javascriptobjectnotation, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	return this.LoadJSON(javascriptobjectnotation)
}

// LoadFile takes a path and attempts to load it with the proper loader based on extension.
func (this *Settings) LoadFile(path string) error {
	switch ext := filepath.Ext(path); ext {
	case ".json":
		return this.LoadJSONFile(path)
	case ".lua":
		return this.LoadLuaFile(path)
	default:
		return fmt.Errorf("Unable to determine config file type for path %s", path)
	}
}

// MergeSettings takes a new map[string]interface{} of settings and merges it into
//  the existing one recursively.
func (this *Settings) MergeSettings(newSettings map[string]interface{}) error {
	return mergeMaps(&this.settings, &newSettings)
}

// mergeMaps takes two maps and combines them, preferring the keys in the newer
//  map.
func mergeMaps(existing, new *map[string]interface{}) error {
	existingmap := *existing
	newmap := *new

	for key, value := range newmap {
		if newvalue, ok := value.(map[string]interface{}); ok {
			if existingvalue, ok := existingmap[key].(map[string]interface{}); ok {
				mergeMaps(&existingvalue, &newvalue)
			} else {
				existingvalue = make(map[string]interface{})
				existingmap[key] = existingvalue
				mergeMaps(&existingvalue, &newvalue)
			}
		} else {
			existingmap[key] = value
		}
	}

	return nil
}

// RawGet will return the interface{} of the value at a specific path, and
//  error if the value cannot be found.
func (this Settings) RawGet(path string) (interface{}, error) {
	parts := strings.Split(path, ":")
	finalpart := parts[len(parts)-1]
	parts = parts[:len(parts)-1]

	node := this.settings

	for _, part := range parts {
		var ok bool
		if node, ok = node[part].(map[string]interface{}); !ok {
			return nil, fmt.Errorf("Could not find %s (missing %s)", path, part)
		}
	}

	if finalpart, ok := node[finalpart].(interface{}); !ok {
		return nil, fmt.Errorf("Could not find %s (missing %s)", path, finalpart)
	} else {
		return finalpart, nil
	}
}

// RawSet will set the value of the config at a specific path. "timid" is used
//  to help describe how to treat values that are part of the path but not the
//  proper type. For instance, given this config:given the path of
//    {"root": {"intermediate": 22}},
//  and this function call:
//    settings.RawSet(timid, "root:intermediate:value", "Hello World")
//  timid == 0 will replace "itermediate" with the required map
//  timid == 1 will instead throw an error claiming  to not be able to find the
//  path.
func (this Settings) RawSet(timid bool, path string, value interface{}) error {
	parts := strings.Split(path, ":")
	finalpart := parts[len(parts)-1]
	parts = parts[:len(parts)-1]

	node := this.settings

	for _, part := range parts {

		// Check if this part exists
		if nodePart, ok := node[part]; ok {
			// Make sure this part isn't anything but another map
			if node, ok = nodePart.(map[string]interface{}); !ok {
				// If timid is true don't overwrite the invalid key
				if timid {
					return fmt.Errorf("Could not find %s (missing %s)", path, part)
				} else {
					// Create empty map for this part
					newMap := make(map[string]interface{})
					node[part] = newMap
					node = newMap
				}
			}
		} else {
			// Create empty map for this part
			newMap := make(map[string]interface{})
			node[part] = newMap
			node = newMap
		}
	}

	node[finalpart] = value
	return nil
}

// Get will retrieve the path and store it inside the interface the best it can.
func (this Settings) Get(path string, target interface{}) error {
	rawvalue, err := this.RawGet(path)
	if err != nil {
		return err
	}

	err = mapstructure.Decode(rawvalue, target)
	return err
}

// GetBool returns a bool stored in the path.
// If the the path isn't defined it will return the defaultValue and an error.
func (this Settings) GetBool(path string, defaultValue bool) (bool, error) {
	rawvalue, err := this.RawGet(path)
	if err != nil {
		return defaultValue, err
	}

	if value, ok := rawvalue.(bool); !ok {
		return defaultValue, fmt.Errorf("%s is not a bool", path)
	} else {
		return value, nil
	}
}

// GetString returns a string stored in the path.
// If the the path isn't defined it will return the defaultValue and an error.
func (this Settings) GetString(path string, defaultValue string) (string, error) {
	rawvalue, err := this.RawGet(path)
	if err != nil {
		return defaultValue, err
	}

	if value, ok := rawvalue.(string); !ok {
		return defaultValue, fmt.Errorf("%s is not a string", path)
	} else {
		return value, nil
	}
}

// GetInt returns a int stored in the path.
// If the the path isn't defined it will return the defaultValue and an error.
func (this Settings) GetInt(path string, defaultValue int64) (int64, error) {
	var target int64

	err := this.Get(path, &target)
	if err != nil {
		return defaultValue, err
	} else {
		return target, nil
	}
}

// GetFloat returns a float stored in the path.
// If the the path isn't defined it will return the defaultValue and an error.
func (this Settings) GetFloat(path string, defaultValue float64) (float64, error) {
	var target float64

	err := this.Get(path, &target)
	if err != nil {
		return defaultValue, err
	} else {
		return target, nil
	}
}
