package main

import (
    "fmt"
    "time"
    "github.com/kamaboko123/tanita_to_fitbit/health_planet"
    "github.com/kamaboko123/tanita_to_fitbit/fitbit"
)

type Syncr struct {
    HealthPlanet *health_planet.Client
    Fitbit *fitbit.Client
}

type AddData struct {
    Date time.Time
    HealthPlanetData *health_planet.InnerscanData
}

func NewSyncr(hp_client *health_planet.Client, fb_client *fitbit.Client) *Syncr {
    return &Syncr{HealthPlanet: hp_client, Fitbit: fb_client}
}

func (s *Syncr) Sync(dry bool) error {
    // get latest data from health planet
    hp_weight, err := s.HealthPlanet.GetInnerscanData()
    if err != nil {
        return err
    }
    Logger.Debug(fmt.Sprintf("Get %d data from Health Planet", len(hp_weight)))
    Logger.Debug(fmt.Sprintf("Latest data: %s", hp_weight))

    var add_data []AddData

    // compare latest data
    for _, hpw := range hp_weight {
        Logger.Debug(fmt.Sprintf("[Health Planet(expect)] %s", hpw))
        
        // この日付のデータがすでにFitbitに存在するか確認
        fb_weight_resp, err := s.Fitbit.GetWeightLog(hpw.Date)
        if err != nil {
            return err
        }
        fb_weight, err := fb_weight_resp.ToWeightLog(s.Fitbit.Timezone)
        if err != nil {
            return err
        }

        Logger.Debug(fmt.Sprintf("[Fitbit(targets)] %s", fb_weight))
        is_exist := false
        for _, fbw := range fb_weight {
            if hpw.Date.Equal(fbw.Date) {
                // 日付が一致した場合はすでにデータが存在しているのでスキップ
                is_exist = true
                break
            }
        }

        if !is_exist {
            add_data = append(add_data, AddData{Date: hpw.Date, HealthPlanetData: hpw})
        }
    }

    fmt.Printf("Found %d new data\n", len(add_data))

    for _, ad := range add_data {
        fmt.Printf("new_data: %s (weight: %fkg, fat: %f%%)", ad.Date, ad.HealthPlanetData.Weight, ad.HealthPlanetData.BodyFat)
        if !dry {
            err = s.Fitbit.CreateWeightAndFatLog(ad.Date, ad.HealthPlanetData.Weight, ad.HealthPlanetData.BodyFat)
            if err != nil {
                fmt.Println(": Failed")
                return err
            }
            fmt.Println(": Success")
        }
        fmt.Printf("\n")
    }

    // add new data to fitbit
    return nil
}

