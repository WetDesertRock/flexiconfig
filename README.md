## FlexiConfig
[![GoDoc](https://godoc.org/github.com/WetDesertRock/flexiconfig?status.svg)](https://godoc.org/github.com/WetDesertRock/flexiconfig)

FlexiConfig is a library for those of us tired of just using a single static JSON or YAML file. It does this by introducing a way to load a lua file as part of the config. The hierarchical nature of FlexiConfig allows you to easily split your config into several files, and even load one as a way of setting defaults. FlexiConfig allows easy access to the values by giving you the ability to access individual sections of the config through a path-like string. This path can be used to set values, retrieve individual values, or even allowing you to pass in a struct to have the config unmarshaled into.

FlexiConfig's featureset is really defined by what I need in my projects. Various features such as file reloading on change and  other programming languages are out of scope for this project. Flexiconfig is really aimed at being simple, hierarchical, and allow you to use a scripting language as a config format.

### Try it out

Say you want to load a config like this:
```json
{
    "Components": {
        "Server": {
            "Port": 2512,
            "Worlds": [
                {
                    "WorldSeed": 241555,
                    "WorldName": "My World"
                },
                {
                    "WorldSeed": 01189998819991197253,
                    "WorldName": "Crowded World"
                }
            ]
        },
        "FileSystem": {
            "Base": "/dev/null"
        }
    }
}
```

Into a go struct like this:
```go
type WorldConfig struct {
    WorldSeed int
    WorldName string
}

type Config struct {
    Port int
    Worlds []WorldConfig
}
```

You can use this code:
```go
// Create a new Settings object
settings := flexiconfig.NewSettings()

// Load JSON file. LoadFile tries to detect what loader to use based on file extension. You can force it with LoadJSONFile
if err := settings.LoadFile("./test.json"); err != nil {
    panic(err)
}

// Or load a JSON byte array:
if err := settings.LoadJSON(json); err != nil {
    panic(err)
}

// Get our WorldConfig struct:
worldConfig := WorldConfig{}
err := settings.Get("Components:Server", &worldConfig)

// Get our filesystem setting, default it to "~"
fsBase, _ := settings.GetString("FileSystem:Base", "~")
```

But what if we want to make this a lua config file?
```lua
function world(seed, name)
    return {
        WorldSeed = seed,
        WorldName = name
    }
end

return {
    Components = {
        Server = {
            Port = 2512,
            Worlds = {
                world(241555, "My World"), -- We can use helper functions!
                world(01189998819991197253, "Crowded World"),  
            }
        },
        FileSystem = {
            Base = "/dev/null", -- Imagine that, using a trailing comma and not having a syntax error??
        }
    }
}
```

If we wanted to load multiple config files we could just run LoadFile (or similar Load function) multiple times. Read the [godoc](https://godoc.org/github.com/WetDesertRock/flexiconfig) for more info. Feel free to read through the example. It covers pretty much the entire library.

## Contributing
Go for it! Note that I'm hesitant to add more features especially if they cause bloat to the library. For instance I don't want to depend on the V8 engine, don't add Javascript support! Feel free to check out the issue tracker, I put things there that I want to get to later.