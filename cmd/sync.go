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

func (s *Syncr) Sync() error {
    // get latest data from health planet
    hp_weight, err := s.HealthPlanet.GetInnerscanData()
    if err != nil {
        return err
    }

    var add_data []AddData

    // compare latest data
    is_exist := false
    for hd, hw := range hp_weight {
        _hd, err := time.Parse("200601021504", hd)
        if err != nil {
            return err
        }
        
        // この日付のデータがすでにFitbitに存在するか確認
        fb_weight, err := s.Fitbit.GetWeightLog(_hd)
        if err != nil {
            return err
        }
        for _, fw := range fb_weight.Weight {
            _fd, err := time.Parse("2006-01-02 15:04:05", fmt.Sprintf("%s %s", fw.Date, fw.Time))
            if err != nil {
                return err
            }

            if _hd == _fd {
                // 日付が一致した場合はすでにデータが存在しているのでスキップ
                is_exist = true
                break
            }
        }

        if !is_exist {
            add_data = append(add_data, AddData{Date: _hd, HealthPlanetData: hw})
        }
    }

    for _, ad := range add_data {
        fmt.Printf("Add data: %s (weight: %fkg, fat: %f%%)\n", ad.Date, ad.HealthPlanetData.Weight, ad.HealthPlanetData.BodyFat)
        err = s.Fitbit.CreateWeightAndFatLog(ad.Date, ad.HealthPlanetData.Weight, ad.HealthPlanetData.BodyFat)
        if err != nil {
            return err
        }
        fmt.Println("Success to add data")
    }

    // add new data to fitbit
    return nil
}

