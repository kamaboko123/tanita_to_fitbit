package main

import (
    "os"
	"fmt"
    "flag"
    "io/ioutil"
    "encoding/json"
    "errors"
    "log/slog"
    "time"
    "github.com/kamaboko123/tanita_to_fitbit/health_planet"
    "github.com/kamaboko123/tanita_to_fitbit/fitbit"
)

const config_file = "config.json"
var Logger *slog.Logger

type config struct {
    HealthPlanet struct {
        ClientId string `json:"client_id"`
        ClientSecret string `json:"client_secret"`
        Timezone string `json:"timezone"`
    } `json:"health_planet"`
    Fitbit struct {
        ClientId string `json:"client_id"`
        ClientSecret string `json:"client_secret"`
        Timezone string `json:"timezone"`
    } `json:"fitbit"`
}

type RunArgs struct {
    mode string
    verbose bool
}

func contains(arr []string, str string) bool {
    for _, a := range arr {
        if a == str {
            return true
        }
    }
    return false
}

func load_config() (*config, error) {
    var c config
    data, err := ioutil.ReadFile(config_file)
    if err != nil {
        return nil, err
    }
    err = json.Unmarshal(data, &c)
    if err != nil {
        return nil, err
    }
    return &c, nil
}


func get_run_args() (*RunArgs,error) {
    m := flag.String("m", "", "mode")
    v := flag.Bool("v", false, "verbose")

    flag.Parse()

    suppport_modes := []string{"sync", "dry-sync", "init_healthplanet", "init_fitbit"}
    if !contains(suppport_modes, *m) {
        return nil, errors.New(fmt.Sprintf("Please set mode with -m. Support modes are %s", suppport_modes))
    }

    return &RunArgs{
        mode: *m,
        verbose: *v,
    }, nil
}

func get_healthplanet_auth(conf config) (*health_planet.Auth) {
    tanita_client_id := conf.HealthPlanet.ClientId
    tanita_client_secret := conf.HealthPlanet.ClientSecret
    hp_auth := health_planet.NewAuth("https://www.healthplanet.jp", tanita_client_id, tanita_client_secret, "hp_token.json", Logger)
    
    return hp_auth
}

func get_fitbit_auth(conf config) (*fitbit.Auth) {
    fitbit_client_id := conf.Fitbit.ClientId
    fitbit_client_secret := conf.Fitbit.ClientSecret
    fb_auth := fitbit.NewAuth("https://api.fitbit.com", fitbit_client_id, fitbit_client_secret, "fb_token.json")

    return fb_auth
}

func run_init_healthplanet(conf config) error {
    hp_auth := get_healthplanet_auth(conf)
    err := hp_auth.InitToken()
    if err != nil {
        return err
    }
    return nil
}

func run_init_fitbit(conf config) error {
    fb_auth := get_fitbit_auth(conf)
    err := fb_auth.InitToken()
    if err != nil {
        return err
    }
    return nil
}

func run_sync(conf config, dry bool) error {
    hp_tz, err := time.LoadLocation(conf.HealthPlanet.Timezone)
    if err != nil {
        return err
    }
    fb_tz, err := time.LoadLocation(conf.Fitbit.Timezone)
    if err != nil {
        return err
    }

    hp_auth := get_healthplanet_auth(conf)
    err = hp_auth.LoadToken()
    if err != nil {
        return err
    }
    err = hp_auth.RefreshToken()
    if err != nil {
        return err
    }
    hp := health_planet.NewClient("https://www.healthplanet.jp", hp_auth, Logger, hp_tz)

    fb_auth := get_fitbit_auth(conf)
    err = fb_auth.LoadToken()
    if err != nil {
        return err
    }
    err = fb_auth.RefreshToken()
    if err != nil {
        return err
    }
    fb := fitbit.NewClient("https://api.fitbit.com", fb_auth, Logger, fb_tz)

    syncr := NewSyncr(hp, fb)
    err = syncr.Sync(dry)
    if err != nil {
        return err
    }

    return nil
}

func main() {
    args, err := get_run_args()
    if err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
    
    log_level := new(slog.LevelVar)
    log_level.Set(slog.LevelWarn)
    log_opts := &slog.HandlerOptions{
        Level: log_level,
    }
    handler := slog.NewTextHandler(os.Stderr, log_opts)
    Logger = slog.New(handler)
    if args.verbose {
        log_level.Set(slog.LevelDebug)
    }

    // Load config
    conf, err := load_config()
    if err != nil {
        Logger.Error(fmt.Sprintf("Load config failed: %s", err))
        os.Exit(2)
    }

    if args.mode == "init_healthplanet" {
        err := run_init_healthplanet(*conf)
        if err != nil {
            Logger.Error(fmt.Sprintf("Init HealthPlanet failed: %s", err))
            os.Exit(10)
        }
    }else if args.mode == "init_fitbit" {
        err := run_init_fitbit(*conf)
        if err != nil {
            Logger.Error(fmt.Sprintf("Init Fitbit failed: %s", err))
            os.Exit(11)
        }
    }else if (args.mode == "sync") {
        err := run_sync(*conf, false)
        if err != nil {
            Logger.Error(fmt.Sprintf("Sync failed: %s", err))
            os.Exit(12)
        }
        fmt.Println("Sync success")
    }else if (args.mode == "dry-sync") {
        err := run_sync(*conf, true)
        if err != nil {
            Logger.Error(fmt.Sprintf("Dry sync failed: %s", err))
            os.Exit(13)
        }
    }
}
