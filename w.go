package main

import (
    "encoding/json"
    "net/http"
    "strings"
    "log"
    "time"
)

type weatherProvider interface {
    temperature(city string) (float64, error) // in Kelvin, naturally
}

type openWeatherMap struct {
    apiKey string
}
type weatherUnderground struct {
    apiKey string
}
type yahooWeather struct {}

type multiWeatherProvider []weatherProvider

func main() {
    
    mw := multiWeatherProvider{
        openWeatherMap{apiKey: "a9bcf4f4899aaab6b7194e3f674f162b"},
        weatherUnderground{apiKey: "2508132ae0c7601a"},
        yahooWeather{},
    }

    http.HandleFunc("/weather/", func(w http.ResponseWriter, r *http.Request) {
        begin := time.Now()
        city := strings.SplitN(r.URL.Path, "/", 3)[2]

        temp, err := mw.temperature(city)

        if err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }

        w.Header().Set("Content-Type", "application/json; charset=utf-8")
        json.NewEncoder(w).Encode(map[string]interface{}{
            "city": city,
            "temp": map[string]interface{}{
                "K" : temp,
                "C" : kelvinToCelsius(temp),
                "F" : kelvinToFahrenheit(temp),
            },
            "took": time.Since(begin).String(),
        })

    })

    http.ListenAndServe(":8080", nil)
}


func (w openWeatherMap) temperature(city string) (float64, error) {
    resp, err := http.Get("http://api.openweathermap.org/data/2.5/weather?APPID=" + w.apiKey + "&q=" + city)
    if err != nil {
        return 0, err
    }

    defer resp.Body.Close()

    var d struct {
        Main struct {
            Kelvin float64 `json:"temp"`
        } `json:"main"`
    }

    if err := json.NewDecoder(resp.Body).Decode(&d); err != nil {
        return 0, err
    }

    log.Printf("openWeatherMap: %s: %.2f", city, d.Main.Kelvin)
    return d.Main.Kelvin, nil
}

func (w weatherUnderground) temperature(city string) (float64, error) {
    resp, err := http.Get("http://api.wunderground.com/api/" + w.apiKey + "/conditions/q/" + city + ".json")
    if err != nil {
        return 0, err
    }

    defer resp.Body.Close()

    var d struct {
        Observation struct {
            Celsius float64 `json:"temp_c"`
        } `json:"current_observation"`
    }

    if err := json.NewDecoder(resp.Body).Decode(&d); err != nil {
        return 0, err
    }

    kelvin := d.Observation.Celsius + 273.15 + 10.0 // +10.0 seems to fix weatherUnderground's bad forecast
    log.Printf("weatherUnderground: %s: %.2f", city, kelvin)
    return kelvin, nil
}

func (w yahooWeather) temperature(city string) (float64, error) {
    apiurl := "https://query.yahooapis.com/v1/public/yql?q=select%20item.condition%20from%20weather.forecast%20where%20woeid%20in%20(select%20woeid%20from%20geo.places(1)%20where%20text%3D%22"+city+"%22)&format=json"
    resp, err := http.Get(apiurl)
    if err != nil {
        return 0, err
    }

    defer resp.Body.Close()

    var d struct {
    Query struct {
        Count int `json:"count"`
        Created time.Time `json:"created"`
        Lang string `json:"lang"`
        Results struct {
            Channel struct {
                Item struct {
                    Condition struct {
                        Code string `json:"code"`
                        Date string `json:"date"`
                        Temp int `json:"temp,string"`
                        Text string `json:"text"`
                    } `json:"condition"`
                } `json:"item"`
            } `json:"channel"`
        } `json:"results"`
    } `json:"query"`
}

    if err := json.NewDecoder(resp.Body).Decode(&d); err != nil {
        return 0, err
    }

    kelvin := fahrenheitToKelvin(float64(d.Query.Results.Channel.Item.Condition.Temp))
    log.Printf("Yahoo! Weather: %s: %.2f", city, kelvin)
    return kelvin, nil
}

func (w multiWeatherProvider) temperature(city string) (float64, error) {
    // Make a channel for temperatures, and a channel for errors.
    // Each provider will push a value into only one.
    temps := make(chan float64, len(w))
    errs := make(chan error, len(w))


    // For each provider, spawn a goroutine with an anonymous function.
    // That function will invoke the temperature method, and forward the response.
    for _, provider := range w {
        go func(p weatherProvider) {
            k, err := p.temperature(city)
            if err != nil {
                errs <- err
                return
            }
            temps <- k
        }(provider)
    }


    sum := 0.0

    // Collect a temperature or an error from each provider.
    for i := 0; i < len(w); i++ {
        select {
        case temp := <-temps:
            sum += temp
        case err := <-errs:
            return 0, err
        }
    }

    return sum / float64(len(w)), nil
}


func kelvinToCelsius(t float64) (float64) {
    return t - 273.15
}

func kelvinToFahrenheit(t float64) (float64) {
    return t * 9/5 - 459.67
}

func fahrenheitToKelvin(t float64) (float64) {
    return (t + 459.67) * 5/9
}