package flexiconfig

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	lua "github.com/yuin/gopher-lua"

	// "github.com/yuin/gluamapper"
	"github.com/mitchellh/mapstructure"
	luajson "layeh.com/gopher-json"
)

type LuaLoader func(L *lua.LState) int

type Settings struct {
	settings   map[string]interface{}
	luaModules map[string]lua.LGFunction
}

func NewSettings() Settings {
	settings := Settings{}
	settings.settings = make(map[string]interface{})
	settings.luaModules = make(map[string]lua.LGFunction)

	return settings
}

func (this Settings) Print() {
	b := this.GetPrettyJSON("", "  ")
	fmt.Println(string(b))
}

func (this Settings) GetPrettyJSON(prefix, indent string) []byte {
	b, err := json.MarshalIndent(this.settings, prefix, indent)
	if err != nil {
		panic(err)
	}
	return b
}

func (this Settings) GetJSON() []byte {
	b, err := json.Marshal(this.settings)
	if err != nil {
		panic(err)
	}
	return b
}

func (this *Settings) AddLuaLoader(name string, loader lua.LGFunction) {
	this.luaModules[name] = loader
}

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

	return this.LoadLuaState(L.Get(-1))
}

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

	return this.LoadLuaState(L.Get(-1))
}

func (this *Settings) LoadLuaState(lv lua.LValue) error {
	// opt := gluamapper.Option{}
	// opt.NameFunc = func(s string) string { return s }
	// opt.TagName = "gluamapper"

	// newSettings := gluamapper.ToGoValue(lv, opt).(map[interface{}]interface{})

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

func (this Settings) Get(path string, target interface{}) error {
	rawvalue, err := this.RawGet(path)
	if err != nil {
		return err
	}

	err = mapstructure.Decode(rawvalue, target)
	return err
}

func (this Settings) GetBool(path string, def bool) (bool, error) {
	rawvalue, err := this.RawGet(path)
	if err != nil {
		return def, err
	}

	if value, ok := rawvalue.(bool); !ok {
		return def, fmt.Errorf("%s is not a bool", path)
	} else {
		return value, nil
	}
}

func (this Settings) GetString(path string, def string) (string, error) {
	rawvalue, err := this.RawGet(path)
	if err != nil {
		return def, err
	}

	if value, ok := rawvalue.(string); !ok {
		return def, fmt.Errorf("%s is not a string", path)
	} else {
		return value, nil
	}
}

func (this Settings) GetInt(path string, def int64) (int64, error) {
	var target int64

	err := this.Get(path, &target)
	if err != nil {
		return def, err
	} else {
		return target, nil
	}
}

func (this Settings) GetFloat(path string, def float64) (float64, error) {
	var target float64

	err := this.Get(path, &target)
	if err != nil {
		return def, err
	} else {
		return target, nil
	}
}
