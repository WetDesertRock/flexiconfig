package settings

import (
	"fmt"
	"io/ioutil"
	"strings"
	"encoding/json"
	"github.com/yuin/gopher-lua"
	"github.com/yuin/gluamapper"
	"github.com/mitchellh/mapstructure"
)

type Settings struct {
	settings map[string]interface{}
}

func NewSettings() Settings {
	settings := Settings{}
	settings.settings = make(map[string]interface{})

	return settings
}

func (this Settings) Print() {
	fmt.Println(this.settings)
}

func (this *Settings) LoadLuaString(code string) error {
	L := lua.NewState()
	defer L.Close()
	if err := L.DoString(code); err != nil {
		return err
	}

	return this.LoadLuaState(L.Get(-1))
}

func (this *Settings) LoadLuaFile(path string) error {
	L := lua.NewState(lua.Options{
		IncludeGoStackTrace: true,
	})
	defer L.Close()
	if err := L.DoFile(path); err != nil {
		return err
	}

	return this.LoadLuaState(L.Get(-1))
}

func (this *Settings) LoadLuaState(lv lua.LValue) error {
	opt := gluamapper.Option{}
	opt.NameFunc = func(s string) string { return s }
	opt.TagName = "gluamapper"

	_newSettings := gluamapper.ToGoValue(lv, opt).(map[interface{}]interface{})
	newSettings := make(map[string]interface{})

	for key, value := range _newSettings {
		switch key := key.(type) {
			case string:
				newSettings[key] = value
		}
	}

	return this.MergeSettings(newSettings)
}

// LoadJSON takes a byte slice, dejsonifys it, then stores the contents in the
//  Settings object.
func (this *Settings) LoadJSON(b []byte) error {
	var newSettings map[string]interface{};
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

// MergeSettings takes a new map[string]interface{} of settings and merges it into
//  the existing one recursively.
func (this *Settings) MergeSettings(newSettings map[string]interface{}) error {
	return mergeMaps(&this.settings, &newSettings)
}

func mergeMaps(existing, new *map[string]interface{}) error {
	existingmap := *existing
	newmap := *new

	for key,value := range newmap {
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
		var ok bool;
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

func (this Settings) Get(path string, target interface{}) (error) {
	rawvalue, err := this.RawGet(path)
	if err != nil {
		return err
	}

	err = mapstructure.Decode(rawvalue, target)
	return err
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
