package main

import (
    "encoding/json"
    "net/http"
    "strings"
    "strconv"
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
type yahooWeather struct {
}
type darkSky struct {
    apiKey string
}


type multiWeatherProvider []weatherProvider

func main() {
    
    mw := multiWeatherProvider{
        openWeatherMap{apiKey : "a9bcf4f4899aaab6b7194e3f674f162b"},
        weatherUnderground{apiKey : "2508132ae0c7601a"},
        yahooWeather{},
        darkSky{apiKey : "1ddf4420f72cf59d18f3948f4af16415"},
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

func (w darkSky) temperature(city string) (float64, error) {
    lat, lon, err := cityToLatLon(city)
    if err != nil {
        return 0, err
    }
    // log.Printf("%s -> %.2f , %.2f", city, lat, lon)
    resp, err := http.Get("https://api.darksky.net/forecast/" + w.apiKey + "/" + strconv.FormatFloat(lat, 'f', -1, 64) + "," + strconv.FormatFloat(lon, 'f', -1, 64))
    var d struct {
        Latitude float64 `json:"latitude"`
        Longitude float64 `json:"longitude"`
        Timezone string `json:"timezone"`
        Offset int `json:"offset"`
        Currently struct {
            Time int `json:"time"`
            Summary string `json:"summary"`
            Icon string `json:"icon"`
            NearestStormDistance int `json:"nearestStormDistance"`
            PrecipIntensity float64 `json:"precipIntensity"`
            PrecipIntensityError float64 `json:"precipIntensityError"`
            PrecipProbability float64 `json:"precipProbability"`
            PrecipType string `json:"precipType"`
            Temperature float64 `json:"temperature"`
            ApparentTemperature float64 `json:"apparentTemperature"`
            DewPoint float64 `json:"dewPoint"`
            Humidity float64 `json:"humidity"`
            WindSpeed float64 `json:"windSpeed"`
            WindBearing int `json:"windBearing"`
            Visibility float64 `json:"visibility"`
            CloudCover float64 `json:"cloudCover"`
            Pressure float64 `json:"pressure"`
            Ozone float64 `json:"ozone"`
        } `json:"currently"`
        Minutely struct {
            Summary string `json:"summary"`
            Icon string `json:"icon"`
            Data []struct {
                Time int `json:"time"`
                PrecipIntensity float64 `json:"precipIntensity"`
                PrecipIntensityError float64 `json:"precipIntensityError"`
                PrecipProbability float64 `json:"precipProbability"`
                PrecipType string `json:"precipType"`
            } `json:"data"`
        } `json:"minutely"`
        Hourly struct {
            Summary string `json:"summary"`
            Icon string `json:"icon"`
            Data []struct {
                Time int `json:"time"`
                Summary string `json:"summary"`
                Icon string `json:"icon"`
                PrecipIntensity float64 `json:"precipIntensity"`
                PrecipProbability float64 `json:"precipProbability"`
                PrecipType string `json:"precipType,omitempty"`
                Temperature float64 `json:"temperature"`
                ApparentTemperature float64 `json:"apparentTemperature"`
                DewPoint float64 `json:"dewPoint"`
                Humidity float64 `json:"humidity"`
                WindSpeed float64 `json:"windSpeed"`
                WindBearing int `json:"windBearing"`
                Visibility float64 `json:"visibility"`
                CloudCover float64 `json:"cloudCover"`
                Pressure float64 `json:"pressure"`
                Ozone float64 `json:"ozone"`
            } `json:"data"`
        } `json:"hourly"`
        Daily struct {
            Summary string `json:"summary"`
            Icon string `json:"icon"`
            Data []struct {
                Time int `json:"time"`
                Summary string `json:"summary"`
                Icon string `json:"icon"`
                SunriseTime int `json:"sunriseTime"`
                SunsetTime int `json:"sunsetTime"`
                MoonPhase float64 `json:"moonPhase"`
                PrecipIntensity float64 `json:"precipIntensity"`
                PrecipIntensityMax float64 `json:"precipIntensityMax"`
                PrecipIntensityMaxTime int `json:"precipIntensityMaxTime,omitempty"`
                PrecipProbability float64 `json:"precipProbability"`
                PrecipType string `json:"precipType,omitempty"`
                TemperatureMin float64 `json:"temperatureMin"`
                TemperatureMinTime int `json:"temperatureMinTime"`
                TemperatureMax float64 `json:"temperatureMax"`
                TemperatureMaxTime int `json:"temperatureMaxTime"`
                ApparentTemperatureMin float64 `json:"apparentTemperatureMin"`
                ApparentTemperatureMinTime int `json:"apparentTemperatureMinTime"`
                ApparentTemperatureMax float64 `json:"apparentTemperatureMax"`
                ApparentTemperatureMaxTime int `json:"apparentTemperatureMaxTime"`
                DewPoint float64 `json:"dewPoint"`
                Humidity float64 `json:"humidity"`
                WindSpeed float64 `json:"windSpeed"`
                WindBearing int `json:"windBearing"`
                Visibility float64 `json:"visibility,omitempty"`
                CloudCover float64 `json:"cloudCover"`
                Pressure float64 `json:"pressure"`
                Ozone float64 `json:"ozone"`
            } `json:"data"`
        } `json:"daily"`
        Flags struct {
            Sources []string `json:"sources"`
            DatapointStations []string `json:"datapoint-stations"`
            IsdStations []string `json:"isd-stations"`
            MadisStations []string `json:"madis-stations"`
            Units string `json:"units"`
        } `json:"flags"`
    }

    if err := json.NewDecoder(resp.Body).Decode(&d); err != nil {
        return 0, err
    }

    kelvin := fahrenheitToKelvin(d.Currently.Temperature)
    log.Printf("DarkSky Weather: %s: %.2f", city, kelvin)
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

func cityToLatLon(city string) (float64, float64, error) {
    resp, err := http.Get("http://maps.googleapis.com/maps/api/geocode/json?address=" + city + "&sensor=false")
    if err != nil {
        return 0, 0, err
    }

    defer resp.Body.Close()

    var d struct {
    Results []struct {
        AddressComponents []struct {
            LongName string `json:"long_name"`
            ShortName string `json:"short_name"`
            Types []string `json:"types"`
        } `json:"address_components"`
        Geometry struct {
            Bounds struct {
                Northeast struct {
                    Lat float64 `json:"lat"`
                    Lng float64 `json:"lng"`
                } `json:"northeast"`
                Southwest struct {
                    Lat float64 `json:"lat"`
                    Lng float64 `json:"lng"`
                } `json:"southwest"`
            } `json:"bounds"`
            Location struct {
                Lat float64 `json:"lat"`
                Lng float64 `json:"lng"`
            } `json:"location"`
            LocationType string `json:"location_type"`
            Viewport struct {
                Northeast struct {
                    Lat float64 `json:"lat"`
                    Lng float64 `json:"lng"`
                } `json:"northeast"`
                Southwest struct {
                    Lat float64 `json:"lat"`
                    Lng float64 `json:"lng"`
                } `json:"southwest"`
            } `json:"viewport"`
        } `json:"geometry"`
        FormattedAddress string `json:"formatted_address"`
        PlaceID string `json:"place_id"`
        Types []string `json:"types"`
    } `json:"results"`
    Status string `json:"status"`
}

    if err := json.NewDecoder(resp.Body).Decode(&d); err != nil {
        return 0, 0, err
    }
    
    return d.Results[0].Geometry.Location.Lat, d.Results[0].Geometry.Location.Lng, nil
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